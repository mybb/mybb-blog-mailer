package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/github"
	"github.com/mmcdole/gofeed"
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

type NewBlogPost struct {
	Title       string
	Summary     string
	Url         string
	PublishedAt time.Time
	Author      string
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func envOrFail(key, message string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	panic(errors.New(message))
}

var (
	port                      = getEnv("BLOG_MAILER_HTTP_PORT", "80")
	secret                    = []byte(getEnv("BLOG_MAILER_GH_HOOK_SECRET", ""))
	mailGunDomain             = envOrFail("BLOG_MAILER_MG_DOMAIN", "MailGun domain is required - please set the 'BLOG_MAILER_MG_DOMAIN' environment variable")
	mailGunApiKey             = envOrFail("BLOG_MAILER_MG_API_KEY", "MailGun API key is required - please set the 'BLOG_MAILER_MG_API_KEY' environment variable")
	mailGunPublicKey          = envOrFail("BLOG_MAILER_MG_PUBLIC_API_KEY", "MailGun public API key is required - please set the 'BLOG_MAILER_MG_PUBLIC_API_KEY' environment variable")
	mailGunMailingListAddress = envOrFail("BLOG_MAILER_MG_MAILING_LIST_ADDRESS", "MailGun mailing list address is required - please set the 'BLOG_MAILER_MG_MAILING_LIST_ADDRESS' environment variable")
	xmlFeedUrl                = envOrFail("BLOG_MAILER_XML_FEED_URL", "XML feed URL is required - please set the 'BLOG_MAILER_XML_FEED_URL' environment variable")
	lastPostFilePath          = getEnv("BLOG_MAILER_LAST_POST_FILE_PATH", "./last_blog_post.txt")

	httpClient = &http.Client{
		Timeout: time.Second * 5,
	}
)

func getLastPostDate() (*time.Time, error) {
	fileContent, err := ioutil.ReadFile(lastPostFilePath)

	if err != nil {
		return nil, err
	}

	if len(fileContent) == 0 {
		return nil, nil
	}

	parsedTime, err := time.Parse(time.RFC3339, string(fileContent))

	if err != nil {
		return nil, err
	}

	return &parsedTime, nil
}

func tryGetNewPost() (*NewBlogPost, error) {
	resp, err := httpClient.Get(xmlFeedUrl)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	feedParser := gofeed.NewParser()

	feed, err := feedParser.Parse(resp.Body)

	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, nil
	}

	lastPostDate, err := getLastPostDate()

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// We only check the most recent post

	mostRecentPost := feed.Items[0]

	if mostRecentPost.PublishedParsed == nil || !mostRecentPost.PublishedParsed.After(*lastPostDate) {
		return nil, nil
	}

	author := ""

	if mostRecentPost.Author != nil {
		author = mostRecentPost.Author.Name
	}

	return &NewBlogPost{
		Title:       mostRecentPost.Title,
		Summary:     mostRecentPost.Description,
		Url:         mostRecentPost.Link,
		PublishedAt: *mostRecentPost.PublishedParsed,
		Author:      author,
	}, nil
}

func sendMailNotification() {
	_, err := tryGetNewPost()

	if err != nil {
		log.Printf("[ERROR] unable to get new blog post: %s\n", err)

		return
	}

	// TODO: Populate template with details of new blog post

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

		// TODO: save the details of the newBlogPost
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
