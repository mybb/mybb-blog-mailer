package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/schema"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mybb/mybb-blog-mailer/config"
)

type SubscriptionService struct {
	config     *config.Config
	templates  *template.Template
	httpClient *http.Client
	decoder    *schema.Decoder
}

type subscribeRequest struct {
	EmailAddress string `schema:"email,required"`
}

func NewSubscriptionService(config *config.Config) (*SubscriptionService, error) {
	templates, err := template.New("").Funcs(template.FuncMap{
		"toPlainText": func(target string) string {
			return bluemonday.StrictPolicy().Sanitize(target)
		},
		"stripUnsafeTags": func(target string) string {
			return bluemonday.UGCPolicy().Sanitize(target)
		},
	}).ParseGlob("./templates/*")

	if err != nil {
		return nil, fmt.Errorf("error initialising templates: %s", err)
	}

	return &SubscriptionService{
		config:    config,
		templates: templates,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
		decoder: schema.NewDecoder(),
	}, nil
}

/// Index handles a request to /, showing the sign-up form.
func (subService *SubscriptionService) Index(w http.ResponseWriter, r *http.Request) {
	subService.templates.ExecuteTemplate(w, "index.html", nil)
}

/// SignUp handles a POST request to /signup, validating the request and subscribing the user to the mailing list.
func (subService *SubscriptionService) SignUp(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		log.Printf("Error parsing form data for subscribe request: %s\n", err)
		http.Redirect(w, r, "/", 301)
		return
	}

	var subRequest subscribeRequest
	err = subService.decoder.Decode(&subRequest, r.PostForm)

	if err != nil {
		log.Printf("Error parsing form data for subscribe request: %s\n", err)
		http.Redirect(w, r, "/", 301)
		return
	}

	// TODO: Validate email with MailGun API, then subscribe the user
}
