package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

var (
	port   = os.Getenv("PORT")
	secret = os.Getenv("GH_HOOK_SECRET")
)

func handleWebHook(w http.ResponseWriter, r *http.Request) {
	log.Printf("headers: %v\n", r.Header)

	_, err := io.Copy(os.Stdout, r.Body)
	if err != nil {
		log.Println(err)
		return
	}
}

func main() {
	http.HandleFunc("/webhook", handleWebHook)

	address := ":" + port

	log.Printf("Starting webhook server on %s", address)

	log.Fatalln(http.ListenAndServe(address, nil))
}
