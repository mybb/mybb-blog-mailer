package main

import (
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
)

var (
	port   = os.Getenv("PORT")
	secret = []byte(os.Getenv("GH_HOOK_SECRET"))
)

func handleWebHook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, secret)
	if err != nil {
		log.Printf("[ERROR] error validating request body: %s\n", err)
		return
	}

	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("[ERROR] could not parse webhook: %s\n", err)
		return
	}

	switch e := event.(type) {
	case *github.PageBuildEvent:
		log.Printf("[DEBUG] received page build event: %v\n", e)

		// TODO: Parse the event and send the email
	default:
		log.Printf("[WARN] unknown event type: %s\n", github.WebHookType(r))
		return
	}
}

func main() {
	http.HandleFunc("/webhook", handleWebHook)

	address := ":" + port

	log.Printf("[DEBUG] starting webhook server: %s", address)

	log.Fatalln(http.ListenAndServe(address, nil))
}
