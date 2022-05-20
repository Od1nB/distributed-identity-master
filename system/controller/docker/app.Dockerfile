
FROM golang:1.17

RUN mkdir /app
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o .

EXPOSE 8080

CMD ["./controller"]