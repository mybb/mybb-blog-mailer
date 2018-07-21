package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"crypto/subtle"
	"log"
	"encoding/gob"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/gorilla/securecookie"

	"github.com/mybb/mybb-blog-mailer/mail"
	"github.com/mybb/mybb-blog-mailer/templating"
)

type SubscriptionService struct {
	mailHandler  mail.Handler
	templates    *template.Template
	httpClient   *http.Client
	sessionStore sessions.Store
	hmacSecret   string
}

type FlashMessages map[string]string

func NewSubscriptionService(mailHandler mail.Handler, hmacSecret string) (*SubscriptionService, error) {
	templates, err := templating.FindAndParseTemplates("./templates", templating.BuildDefaultFunctionMap())

	if err != nil {
		return nil, fmt.Errorf("error initialising templates: %s", err)
	}

	gob.Register(&FlashMessages{})

	return &SubscriptionService{
		mailHandler: mailHandler,
		templates:   templates,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
		sessionStore: sessions.NewCookieStore(securecookie.GenerateRandomKey(32)), // TODO: do not hardcode session key
		hmacSecret: hmacSecret,
	}, nil
}

/// Index handles a request to /, showing the sign-up form.
func (subService *SubscriptionService) Index(w http.ResponseWriter, r *http.Request) {
	session, err := subService.sessionStore.Get(r, "blog-mailer-session")
	if err != nil {
		log.Printf("[ERROR] getting session for request: %s\n", err)

		http.Error(w, fmt.Sprintf("Error getting session for request: %s", err),
			http.StatusInternalServerError)

		return
	}

	errors := make(FlashMessages)

	flashes := session.Flashes()
	if len(flashes) > 0 {
		if decodedErrors, ok := flashes[0].(*FlashMessages); !ok {
			// Handle the case that it's not an expected type
			log.Printf("[ERROR] decoding flash values for request: %s\n", err)

			http.Error(w, fmt.Sprintf("Error decoding flash values for request: %s", err),
				http.StatusInternalServerError)

			return
		} else {
			errors = *decodedErrors
		}
	}

	subService.templates.ExecuteTemplate(w, "index.html", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
		"errors": errors,
	})
}

/// SignUp handles a POST request to /signup, validating the request and subscribing the user to the mailing list.
func (subService *SubscriptionService) SignUp(w http.ResponseWriter, r *http.Request) {
	session, err := subService.sessionStore.Get(r, "blog-mailer-session")
	if err != nil {
		log.Printf("[ERROR] getting session for request: %s\n", err)

		http.Error(w, fmt.Sprintf("Error getting session for request: %s", err),
			http.StatusInternalServerError)

		return
	}

	err = r.ParseForm()

	if err != nil {
		log.Printf("[ERROR] parsing form data for subscribe request: %s\n", err)

		http.Error(w, fmt.Sprintf("Error parsing form data for subscribe request: %s", err),
			http.StatusInternalServerError)
		return
	}

	name, ok := r.PostForm["name"]

	if !ok || len(name) == 0 || len(name[0]) == 0 {
		session.AddFlash(FlashMessages{
			"error": "Name is required",
		})

		if err = session.Save(r, w); err != nil {
			log.Printf("[ERROR] saving session data: %s\n", err)

			http.Error(w, fmt.Sprintf("Error saving session data: %s", err),
				http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", 301)
		return
	}

	emailAddress, ok := r.PostForm["email"]

	if !ok || len(emailAddress) == 0 || len(emailAddress[0]) == 0 {
		session.AddFlash(FlashMessages{
			"error": "Email address is required",
		})

		if err = session.Save(r, w); err != nil {
			log.Printf("[ERROR] saving session data: %s\n", err)

			http.Error(w, fmt.Sprintf("Error saving session data: %s", err),
				http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", 301)
		return
	}

	isValidEmail, err := subService.mailHandler.CheckValidEmail(emailAddress[0])

	if err != nil || !isValidEmail {
		if err != nil {
			log.Printf("[ERROR] parsing email address '%s': %s\n", emailAddress, err)

			http.Error(w, fmt.Sprintf("Error parsing email address: %s", err),
				http.StatusInternalServerError)

			return
		}

		session.AddFlash(FlashMessages{
			"error": "Invalid email address",
		})

		if err = session.Save(r, w); err != nil {
			log.Printf("[ERROR] saving session data: %s\n", err)

			http.Error(w, fmt.Sprintf("Error saving session data: %s", err),
				http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", 301)
		return
	}

	err = subService.sendEmailSubscriptionConfirmation(emailAddress[0], name[0])

	if err != nil {
		log.Printf("[ERROR] sending subscription confirmation email to '%s': %s\n", emailAddress, err)

		session.AddFlash(FlashMessages{
			"error": "Failed to send subscription confirmation email",
		})

		if err = session.Save(r, w); err != nil {
			log.Printf("[ERROR] saving session data: %s\n", err)

			http.Error(w, fmt.Sprintf("Error saving session data: %s", err),
				http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", 301)
		return
	}

	subService.templates.ExecuteTemplate(w, "signup.html", map[string]interface{}{
		"name": name[0],
		"emailAddress": emailAddress[0],
	})
}

func (subService *SubscriptionService) sendEmailSubscriptionConfirmation(emailAddress, name string) error {
	var plainTextContentBuffer bytes.Buffer
	var htmlContentBuffer bytes.Buffer

	token := subService.generateEmailConfirmationToken(emailAddress, name)

	err := subService.templates.ExecuteTemplate(&plainTextContentBuffer, "emails/confirm_subscription.txt", map[string]string{
		"emailAddress": emailAddress,
		"name": name,
		"token": token,
	})

	if err != nil {
		return err
	}

	err = subService.templates.ExecuteTemplate(&htmlContentBuffer, "emails/confirm_subscription.html", map[string]string{
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

func (subService *SubscriptionService) ConfirmSignUp(w http.ResponseWriter, r *http.Request) {
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