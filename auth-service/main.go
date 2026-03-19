package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var (
	db            *sql.DB
	errDBNotReady = errors.New("db not initialized")
	queryUserID   = func(username, password string) (int, error) {
		if db == nil {
			return 0, errDBNotReady
		}

		var id int
		err := db.QueryRow("SELECT id FROM users WHERE username = $1 AND password = $2", username, password).Scan(&id)
		return id, err
	}
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var u User
	json.NewDecoder(r.Body).Decode(&u)

	if db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err := db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", u.Username, u.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var u User
	json.NewDecoder(r.Body).Decode(&u)

	_, err := queryUserID(u.Username, u.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"token": "dummy-jwt-token-for-" + u.Username})
}

func main() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, username TEXT UNIQUE, password TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	db.Exec("INSERT INTO users (username, password) VALUES ('admin', 'password') ON CONFLICT DO NOTHING")

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	
	log.Println("Auth service running on port 8081")
	http.ListenAndServe(":8081", nil)
}