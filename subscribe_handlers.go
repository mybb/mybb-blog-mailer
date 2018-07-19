package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/schema"
	"github.com/microcosm-cc/bluemonday"

	"github.com/mybb/mybb-blog-mailer/mail"
)

type SubscriptionService struct {
	mailHandler mail.Handler
	templates   *template.Template
	httpClient  *http.Client
	decoder     *schema.Decoder
}

type subscribeRequest struct {
	EmailAddress string `schema:"email,required"`
}

func NewSubscriptionService(mailHandler mail.Handler) (*SubscriptionService, error) {
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
		mailHandler: mailHandler,
		templates:   templates,
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
		http.Error(w, fmt.Sprintf("Error parsing form data for subscribe request: %s", err), 500)
		return
	}

	var subRequest subscribeRequest
	err = subService.decoder.Decode(&subRequest, r.PostForm)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing form data for subscribe request: %s", err), 500)
		return
	}

	isValidEmail, err := subService.mailHandler.CheckValidEmail(subRequest.EmailAddress)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error validating email: %s", err), 500)
		return
	}

	if !isValidEmail {
		http.Redirect(w, r, "/", 301)
		// TODO: Flash message about invalid email?
		return
	}

	// TODO: subscribe the user to the mailing list
}
