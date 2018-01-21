package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"gopkg.in/mailgun/mailgun-go.v1"
)

const (
	emailContent = `Hi, %recipient_email%

There has been a new entry posted on the MyBB blog. Pleas click here to view it (TODO).

Unsubscribe: %unsubscribe_url%
Mailing list unsubscribe: %mailing_list_unsubscribe_url%
Unsubscribe email: %unsubscribe_email%`
)

var (
	port             = os.Getenv("PORT")
	secret           = []byte(os.Getenv("GH_HOOK_SECRET"))
	mailGunDomain    = os.Getenv("MAILGUN_DOMAIN")
	mailGunApiKey    = os.Getenv("MAILGUN_API_KEY")
	mailGunPublicKey = os.Getenv("MAILGUN_PUBLIC_KEY")
)

func sendMailNotification() {
	mg := mailgun.NewMailgun(mailGunDomain, mailGunApiKey, mailGunPublicKey)

	message := mg.NewMessage(
		"no-reply@mybbstuff.com",
		"New MyBB Blog Post",
		emailContent,
		"blog.mybb.com@mybbstuff.com")

	resp, id, err := mg.Send(message)
	if err != nil {
		log.Printf("[ERROR] unable to send update email: %s\n", err)
	}

	log.Printf("[DEBUG] sent email with id %s: %s\n", id, resp)
}

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
		switch buildStatus := e.Build.GetStatus(); buildStatus {
		case "built":
			log.Println("[DEBUG] received successful page build event, reading feed to send emails")

			// Build was successful, so get the newest post and send email via MailGun
			sendMailNotification()
		case "queued":
			log.Println("[DEBUG] rceived page build event with queued status")
		case "building":
			log.Println("[DEBUG] rceived page build event with building status")
		case "errored":
			buildError := e.Build.GetError()
			buildErrorMessage := ""

			if buildError != nil {
				buildErrorMessage = buildError.GetMessage()
			}

			if len(buildErrorMessage) > 0 {
				log.Printf("[WARN] received page build event with error message: %s\n", buildErrorMessage)
			} else {
				log.Println("[WARN] received page build event with error status but no error message")
			}
		default:
			log.Printf("[WARN] received page build event with unknown build status: %s\n", buildStatus)
		}
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
