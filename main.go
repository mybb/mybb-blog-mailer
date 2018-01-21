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
	emailContentPlain = `Hi, %recipient_email%

There has been a new entry posted on the MyBB blog. Pleas click here to view it (TODO).

Unsubscribe from MyBB blog updates: %mailing_list_unsubscribe_url%`

	emailContentHtml = `<div class="container">
	<p>Hi, <stong>%recipient_email%</strong></p>

	<p>There has been a new entry posted on the MyBB blog. Pleas click here to view it (TODO).</p>

	<p class="footer"><a class="btn btn--unsubscribe" href="%mailing_list_unsubscribe_url%">Unsubscribe from MyBB blog updates</a></p>
</div>`
)

var (
	port                      = os.Getenv("PORT")
	secret                    = []byte(os.Getenv("GH_HOOK_SECRET"))
	mailGunDomain             = os.Getenv("MG_DOMAIN")
	mailGunApiKey             = os.Getenv("MG_API_KEY")
	mailGunPublicKey          = os.Getenv("MG_PUBLIC_API_KEY")
	mailGunMailingListAddress = os.Getenv("MG_MAILING_LIST_ADDRESS")
)

func sendMailNotification() {
	// TODO: Get the most recent post from the XML feed, and check if we've already sent a message for it. If so, don't send an email

	mg := mailgun.NewMailgun(mailGunDomain, mailGunApiKey, mailGunPublicKey)

	message := mg.NewMessage(
		mailGunMailingListAddress,
		"New MyBB Blog Post",
		emailContentPlain,
		mailGunMailingListAddress)

	message.SetHtml(emailContentHtml)
	message.AddHeader("List-Unsubscribe", "%unsubscribe_email%")

	resp, id, err := mg.Send(message)
	if err != nil {
		log.Printf("[ERROR] unable to send update email: %s\n", err)
	} else {
		log.Printf("[DEBUG] sent email with id %s and status: %s\n", id, resp)
	}
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
