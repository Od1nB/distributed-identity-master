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

// type App struct {
// 	client       *acapy.Client
// 	server       *http.Server
// 	ledgerURL    string
// 	port         int
// 	label        string
// 	seed         string
// 	rand         string
// 	myDID        string
// 	connectionID string
// }

type Credentials struct {
	Password string `json:"password" db:"password_hash"`
	Username string `json:"username" db:"username"`
	Wallet   string `json:"wallet" db:"wallet"`
}

type Post struct {
	ImageUrl      string `json:"image_url"`
	KeyManagment  string `json:"managed"`
	Label         string `json:"label"`
	WalletDisType string `json:"wallet_dispatch_type"`
	WalletKey     string `json:"wallet_key"`
	WalletName    string `json:"MyNewWallet"`
	WalletType    string `json:"wallet_type"`
	WalletWebHook string `json:"wallet_webhook_urls"`
}

var db *sql.DB

func main() {
	var err error

	// fmt.Println("here")
	// c := acapy.NewClient(fmt.Sprintf("http://localhost:%d", 8001))
	// fmt.Println(c)

	// const (
	// 	host     = "localhost"
	// 	port     = 5432
	// 	user     = "postgres"
	// 	password = "password"
	// 	dbname   = "controller_db"
	// )
	http.HandleFunc("/signup", Signup)
	http.HandleFunc("/signin", SignIn)
	http.HandleFunc("/init", OneUser)
	http.HandleFunc("/test", Ti)
	http.HandleFunc("/addwallet", createWallet)
	http.HandleFunc("/initdb", initDB)

	db, err = sql.Open("postgres",
		"host=system-controller-db-1 user=postgres password=password port=5432 dbname=controller_db sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	fmt.Println("after init")
	connErr := db.Ping()
	if connErr != nil {
		fmt.Println("cant connect to db")
	}
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}

}

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
	fmt.Println(db)
}

func Ti(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "AAAAight, this is what i have for ya")
}

func createWallet(w http.ResponseWriter, r *http.Request) {
	fmt.Print("making new wallet")
	vals := Post{
		ImageUrl:      "https://aries.ca/images/sample.png",
		KeyManagment:  "managed",
		Label:         "Alice",
		WalletDisType: "default",
		WalletKey:     "MySecretKey123",
		WalletName:    "MyNewWallet",
		WalletType:    "indy",
		WalletWebHook: "http://localhost:8022/webhooks"}
	json_data, err := json.Marshal(vals)
	if err != nil {
		panic(err)
	}
	fmt.Println("sending post")
	resp, err := http.Post(
		"http://org1-agent:8001/multitenancy/wallet",
		"application/json",
		bytes.NewBuffer(json_data))
	if err != nil {
		fmt.Println("got this response")
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println(resp)
	fmt.Println(err)
}

func OneUser(w http.ResponseWriter, r *http.Request) {
	hashedPass, _ := bcrypt.GenerateFromPassword([]byte("password"), 3)
	db.Query("insert into userinfo ($1, $2)", "user1", string(hashedPass))
}

func Signup(w http.ResponseWriter, r *http.Request) {
	//Parse data into Creds instance
	fmt.Println("got signup req")
	creds := &Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	fmt.Println(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 3)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println(hashedPass)
	if _, err = db.Query("insert into userinfo values ($1, $2)", creds.Username, string(hashedPass)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//TODO: Make wallet for user in the ACAPY and write wallet handle to db
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	//Parse incoming req to login into Credential struct
	fmt.Println("got signin req")
	creds := &Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Println(creds)
	db_password := db.QueryRow("select password_hash from userinfo where username=$1", creds.Username)
	fmt.Println(db_password)

	storedCreds := &Credentials{}
	err = db_password.Scan(&storedCreds.Password)
	fmt.Println("error", err)
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
	}

	//TODO: Fetch from ACAPY wallet and return in body

}
