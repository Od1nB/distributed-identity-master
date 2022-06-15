package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type CreatedWallet struct {
	CreatedAt        string   `json:"created_at"`
	KeyManagmentMode string   `json:"key_management_mode"`
	Settings         struct{} `json:"settings"`
	State            string   `json:"state"`
	Token            string   `json:"token"`
	UpdatedAt        string   `json:"updated_at"`
	WalletId         string   `json:"wallet_id"`
}

type Credentials struct {
	Password string `json:"password" db:"password_hash"`
	Username string `json:"username" db:"username"`
	Wallet   string `json:"wallet" db:"wallet"`
}

type NewWalletPost struct {
	ImageUrl      string `json:"image_url"`
	KeyManagment  string `json:"managed"`
	Label         string `json:"label"`
	WalletDisType string `json:"wallet_dispatch_type"`
	WalletKey     string `json:"wallet_key"`
	WalletName    string `json:"wallet_name"`
	WalletType    string `json:"wallet_type"`
	WalletWebHook string `json:"wallet_webhook_urls"`
}

func setDefWalletPost(p *NewWalletPost) {
	p.ImageUrl = "https://aries.ca/images/sample.png"
	p.KeyManagment = "managed"
	p.WalletDisType = "default"
	p.WalletType = "indy"
	p.WalletWebHook = "http://localhost:8022/webhooks"
}

var db *sql.DB

func main() {
	var err error
	http.HandleFunc("/signup", Signup)
	http.HandleFunc("/signin", SignIn)
	http.HandleFunc("/initdb", initDB)

	db, err = sql.Open("postgres",
		"host=system-controller-db-1 user=postgres password=password port=5432 dbname=controller_db sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	connErr := db.Ping()
	if connErr != nil {
		fmt.Println("cant connect to db")
	}
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}

}

// In case the initial connect does not work
func initDB(w http.ResponseWriter, r *http.Request) {
	var err error
	db, err = sql.Open("postgres",
		"host=system-controller-db-1 user=postgres password=password port=5432 dbname=controller_db sslmode=disable")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = db.Ping()
	if err != nil {
		fmt.Println("failed on ping")
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hashedPass, _ := bcrypt.GenerateFromPassword([]byte("password"), 3)
	db.Query("insert into userinfo ($1, $2)", "user1", string(hashedPass))
}

func Signup(w http.ResponseWriter, r *http.Request) {
	//Parse data into Creds instance
	creds := &Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 3)
	if err != nil {
		fmt.Println("could not create password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err = db.Query("insert into userinfo values ($1, $2)", creds.Username, string(hashedPass)); err != nil {
		fmt.Println("could not insert into db")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	vals := &NewWalletPost{}
	setDefWalletPost(vals)

	vals.Label = creds.Username
	vals.WalletKey = string(hashedPass)
	vals.WalletName = creds.Username

	json_data, _ := json.Marshal(vals)

	resp, err := http.Post(
		"http://org1-agent:8001/multitenancy/wallet",
		"application/json",
		bytes.NewBuffer(json_data))
	if err != nil {
		fmt.Println(err)
		// w.WriteHeader(http.StatusInternalServerError)
	}

	cwall := &CreatedWallet{}
	err = json.NewDecoder(resp.Body).Decode(cwall)
	if err != nil {
		fmt.Println(err)
	}

	if _, err = db.Exec("update userinfo set wallet =$2 where username = $1", creds.Username, cwall.WalletId); err != nil {
		fmt.Println("could not insert into db")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type SigninResp struct {
	Wallet string `json:"wallet_id"`
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	//Parse incoming req to login into Credential struct
	creds := &Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	db_password := db.QueryRow("select password_hash from userinfo where username=$1", creds.Username)
	storedCreds := &Credentials{}
	err = db_password.Scan(&storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	wallet_ID := db.QueryRow("select wallet from userinfo where username=$1", creds.Username)
	err = wallet_ID.Scan(&storedCreds.Wallet)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	confirmedWallet := SigninResp{Wallet: storedCreds.Wallet}
	json_resp, _ := json.Marshal(confirmedWallet)
	w.Write(json_resp)

}
