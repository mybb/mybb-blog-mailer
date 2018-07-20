package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/gorilla/csrf"

	"github.com/mybb/mybb-blog-mailer/mail"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"crypto/subtle"
)

type SubscriptionService struct {
	mailHandler mail.Handler
	templates   *template.Template
	httpClient  *http.Client
	hmacSecret string
}

func NewSubscriptionService(mailHandler mail.Handler, hmacSecret string) (*SubscriptionService, error) {
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
		hmacSecret: hmacSecret,
	}, nil
}

/// Index handles a request to /, showing the sign-up form.
func (subService *SubscriptionService) Index(w http.ResponseWriter, r *http.Request) {
	subService.templates.ExecuteTemplate(w, "index.html", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
	})
}

/// SignUp handles a POST request to /signup, validating the request and subscribing the user to the mailing list.
func (subService *SubscriptionService) SignUp(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing form data for subscribe request: %s", err), 500)
		return
	}

	emailAddress := r.PostForm.Get("email")
	name := r.PostForm.Get("name")

	if len(emailAddress) == 0 || len(name) == 0 {
		http.Error(w, "No emaila ddress or name provided", 500)
		return
	}

	isValidEmail, err := subService.mailHandler.CheckValidEmail(emailAddress)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error validating email: %s", err), 500)
		return
	}

	if !isValidEmail {
		http.Redirect(w, r, "/", 301)
		// TODO: Flash message about invalid email?
		return
	}

	err = subService.sendEmailSubscriptionConfirmation(emailAddress, name)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error sending confirmation email: %s", err), 500)
		return
	}
}

func (subService *SubscriptionService) sendEmailSubscriptionConfirmation(emailAddress, name string) error {
	var plainTextContentBuffer bytes.Buffer
	var htmlContentBuffer bytes.Buffer

	token := subService.generateEmailConfirmationToken(emailAddress, name)

	err := subService.templates.ExecuteTemplate(&plainTextContentBuffer, "confirm_subscription_email.tmpl", map[string]string{
		"emailAddress": emailAddress,
		"name": name,
		"token": token,
	})

	if err != nil {
		return err
	}

	err = subService.templates.ExecuteTemplate(&htmlContentBuffer, "confirm_subscription_email.html", map[string]string{
		"emailAddress": emailAddress,
		"name": name,
		"token": token,
	})

	if err != nil {
		return err
	}

	err = subService.mailHandler.SendSubscriptionConfirmationEmail(emailAddress,
		plainTextContentBuffer.String(), htmlContentBuffer.String())

	return err
}

func (subService *SubscriptionService) generateEmailConfirmationToken(emailAddress, name string) string {
	message := emailAddress + "_" + name

	key := []byte(subService.hmacSecret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))

	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func (subService *SubscriptionService) ConfirmSignup(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	emailAddress, ok := query["emailAddress"]

	if !ok || len(emailAddress) == 0 {
		http.Error(w, "Email address parameter missing", 400)
		return
	}

	name, ok := query["name"]

	if !ok || len(name) == 0 {
		http.Error(w, "Name parameter missing", 400)
		return
	}

	token, ok := query["token"]

	if !ok || len(token) == 0 {
		http.Error(w, "Token parameter missing", 400)
		return
	}

	expectedToken := subService.generateEmailConfirmationToken(emailAddress[0], name[0])

	if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(token[0])) != 1 {
		http.Error(w, "Invalid token", 400)
		return
	}

	err := subService.mailHandler.SubscribeEmailToMailingList(emailAddress[0], name[0])

	if err != nil {
		http.Error(w, fmt.Sprintf("Error subscribing to mailing list: %s", err), 400)
		return
	}
}