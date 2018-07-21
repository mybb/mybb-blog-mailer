package main

import (
	"net/http"
	"time"
	"fmt"
	"log"
	"bytes"
	"io/ioutil"
	"os"
	"html/template"

	"github.com/google/go-github/github"
	"github.com/mmcdole/gofeed"

	"github.com/mybb/mybb-blog-mailer/mail"
)

type WebHookService struct {
	mailHandler   mail.Handler
	templates *template.Template
	httpClient    *http.Client
	webHookSecret []byte
	xmlFeedUrl    string
	lastPostDateFilePath string
}

type newBlogPost struct {
	Title       string
	Summary     string
	Url         string
	PublishedAt time.Time
	Author      string
}

func NewWebHookService(mailHandler mail.Handler, templates *template.Template, webHookSecret string, xmlFeedUrl string,
	lastPostDateFilePath string) (*WebHookService) {
	return &WebHookService{
		mailHandler: mailHandler,
		templates: templates,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
		webHookSecret: []byte(webHookSecret),
		xmlFeedUrl:    xmlFeedUrl,
		lastPostDateFilePath: lastPostDateFilePath,
	}
}

/// Index handles a request to /webhook, handling an incoming web hook request from GitHub.
func (whService *WebHookService) Index(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, whService.webHookSecret)

	if err != nil {
		errorMessage := fmt.Sprintf("error validating request body: %s", err)

		log.Printf("[ERROR] " + errorMessage + "\n")

		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		errorMessage := fmt.Sprintf("could not parse webhook: %s", err)

		log.Printf("[ERROR] " + errorMessage + "\n")

		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.PingEvent:
		log.Println("[DEBUG] received ping event")
	case *github.PageBuildEvent:
		switch buildStatus := e.Build.GetStatus(); buildStatus {
		case "built":
			log.Println("[DEBUG] received successful page build event, reading feed to send emails")

			// Build was successful, so get the newest post and send email
			whService.sendMailNotification()
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
				log.Printf("[WARN] received page build event with error message: %s\n")
			} else {
				log.Println("[WARN] received page build event with error status but no error message")
			}
		default:
			log.Printf("[WARN] received page build event with unknown build status: %s\n", buildStatus)
		}
	default:
		warningMessage := fmt.Sprintf("unknown event type: %s", github.WebHookType(r))

		log.Printf("[WARN] " + warningMessage + "\n")

		http.Error(w, warningMessage, http.StatusNotImplemented)
		return
	}
}

func (whService *WebHookService) getLastPostDate() (*time.Time, error) {
	fileContent, err := ioutil.ReadFile(whService.lastPostDateFilePath)

	if os.IsNotExist(err) {
		currentTime := time.Now()

		ioutil.WriteFile(whService.lastPostDateFilePath, []byte(currentTime.Format(time.RFC3339)), 0644)

		return &currentTime, nil
	}

	if err != nil || len(fileContent) == 0 {
		return nil, err
	}

	parsedTime, err := time.Parse(time.RFC3339, string(fileContent))

	if err != nil {
		return nil, err
	}

	return &parsedTime, nil
}

func (whService *WebHookService) tryGetNewPost() (*newBlogPost, error) {
	resp, err := whService.httpClient.Get(whService.xmlFeedUrl)

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

	lastPostDate, err := whService.getLastPostDate()

	if err != nil {
		return nil, err
	}

	// We only check the most recent post

	mostRecentPost := feed.Items[0]

	if lastPostDate != nil && (mostRecentPost.PublishedParsed == nil || !mostRecentPost.PublishedParsed.After(*lastPostDate)) {
		return nil, nil
	}

	author := ""

	if mostRecentPost.Author != nil {
		author = mostRecentPost.Author.Name
	}

	return &newBlogPost{
		Title:       mostRecentPost.Title,
		Summary:     mostRecentPost.Description,
		Url:         mostRecentPost.Link,
		PublishedAt: *mostRecentPost.PublishedParsed,
		Author:      author,
	}, nil
}

func (whService *WebHookService) sendMailNotification() {
	newBlogPost, err := whService.tryGetNewPost()

	if err != nil {
		log.Printf("[ERROR] unable to get new blog post: %s\n", err)

		return
	}

	if newBlogPost == nil {
		log.Println("[DEBUG] no new blog post found")

		return
	} else {
		log.Printf("[DEBUG] found new blog post: %+v\n", *newBlogPost)
	}

	var plainTextContentBuffer bytes.Buffer

	err = whService.templates.ExecuteTemplate(&plainTextContentBuffer, "emails/blog_post_notification.txt", newBlogPost)

	if err != nil {
		log.Printf("[ERROR] unable to create plaintext email content: %s\n", err)

		return
	}

	var htmlContentBuffer bytes.Buffer

	err = whService.templates.ExecuteTemplate(&htmlContentBuffer, "emails/blog_post_notification.html", newBlogPost)

	if err != nil {
		log.Printf("[ERROR] unable to create HTML email content: %s\n", err)

		return
	}

	err = whService.mailHandler.SendNotificationToMailingList(newBlogPost.Title, plainTextContentBuffer.String(),
		htmlContentBuffer.String())

	if err != nil {
		log.Printf("[ERROR] sending blog post notification for post '%s': %s", newBlogPost.Title, err)
	} else {
		lastPostDate := newBlogPost.PublishedAt.Format(time.RFC3339)

		err = ioutil.WriteFile(whService.lastPostDateFilePath, []byte(lastPostDate), 0644)

		if err != nil {
			log.Printf("[WARN] saving last post date '%s' for '%s': %s\n", lastPostDate, newBlogPost.Title, err)
		}
	}
}