package main

import (
	"fmt"
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

		http.Error(w, "error validating request body", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("[ERROR] could not parse webhook: %s\n", err)

		http.Error(w, "error parsing webhook content", http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.PageBuildEvent:
		buildError := e.Build.GetError()
		buildErrorMessage := ""

		if buildError != nil {
			buildErrorMessage = buildError.GetMessage()
		}

		if len(buildErrorMessage) > 0 {
			log.Printf("[WARN] received page build event with error message: %s\n", buildErrorMessage)
			return
		}

		// Build was successful, so get the newest post and send email via MailGun
	default:
		warningMessage := fmt.Sprintf("unknown event type: %s", github.WebHookType(r))

		log.Printf("[WARN] "+warningMessage+"\n", github.WebHookType(r))

		http.Error(w, warningMessage, http.StatusNotImplemented)
		return
	}
}

func main() {
	http.HandleFunc("/webhook", handleWebHook)

	address := ":" + port

	log.Printf("[DEBUG] starting webhook server: %s", address)

	log.Fatalln(http.ListenAndServe(address, nil))
}
