package main

import (
	"log"
	"net/http"
	"os"

	legalmail "github.com/example/legal-matter-mailer"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	sender := legalmail.NewReceiptSender(legalmail.NewInfraiEmail(key))
	server := &http.Server{Addr: addr, Handler: legalmail.NewHandler(sender)}
	log.Printf("legal mail service listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
