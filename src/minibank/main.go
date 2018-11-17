package main

import (
	"minibank/handlers"
	"minibank/models"
	"net/http"
	"os"
	"log"
)

func main() {
	// Connect to the database

	dbDoneCh := make(chan bool)
	dbDone := false

	cdbDoneCh := make(chan bool)
	cdbDone := false

	go models.InitDB(dbDoneCh)
	defer models.Database.Close()

	if models.CassandraEnabled {
		log.Print("Calling InitCassandra")
		go models.InitCassandra(cdbDoneCh)
		defer models.CassandraSession.Close()
	}

	log.Print("Calling HandleFunc")
	http.HandleFunc("/api/account/register", validateDBConn(handlers.RegisterHandler, &dbDone, &cdbDone))
	http.HandleFunc("/api/account/login", validateDBConn(handlers.LoginHandler, &dbDone, &cdbDone))
	http.HandleFunc("/api/account/token", validateDBConn(handlers.TokenHandler, &dbDone, &cdbDone))
	http.HandleFunc("/api/account/sessions", handlers.AuthValidationMiddleware(handlers.SessionListHandler))

	log.Print("updateding DB for mysql")
	go updateDBDone(&dbDone, dbDoneCh)
	log.Print("updateding DB for cassandra db")
	go updateDBDone(&cdbDone, cdbDoneCh)
	http.ListenAndServe(port(), nil)
}

func validateDBConn(next http.HandlerFunc, dbDone *bool, cdbDone *bool) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *dbDone || *cdbDone {
			next(w, r)
		} else {
			handlers.ServerUnavailableHandler(w, r)
		}
	})
}

func updateDBDone(dbdone *bool, dbDoneCh <-chan bool) {
	*dbdone = <-dbDoneCh
}

// port looks up service listening port
func port() string {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	return ":" + port
}
