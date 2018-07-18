package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/github"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"gopkg.in/mailgun/mailgun-go.v1"
	"encoding/base64"
	"crypto/aes"
	"crypto/cipher"
	"io"
	"crypto/rand"
)

type newBlogPost struct {
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

func ensureEncryptionKeyLength(key string) []byte {
	keyBytes := []byte(key)

	if len(keyBytes) != 32 {
		panic(fmt.Errorf("invalid encryption key length %d, must be 32 bytes long", len(keyBytes)))
	}

	return keyBytes
}

var (
	port                      = getEnv("BLOG_MAILER_HTTP_PORT", "80")
	secret                    = []byte(getEnv("BLOG_MAILER_GH_HOOK_SECRET", ""))
	mailGunDomain             = envOrFail("BLOG_MAILER_MG_DOMAIN", "mailgun domain is required - please set the 'BLOG_MAILER_MG_DOMAIN' environment variable")
	mailGunApiKey             = envOrFail("BLOG_MAILER_MG_API_KEY", "mailgun API key is required - please set the 'BLOG_MAILER_MG_API_KEY' environment variable")
	mailGunPublicKey          = envOrFail("BLOG_MAILER_MG_PUBLIC_API_KEY", "mailgun public API key is required - please set the 'BLOG_MAILER_MG_PUBLIC_API_KEY' environment variable")
	mailGunMailingListAddress = envOrFail("BLOG_MAILER_MG_MAILING_LIST_ADDRESS", "mailgun mailing list address is required - please set the 'BLOG_MAILER_MG_MAILING_LIST_ADDRESS' environment variable")
	aesKey = ensureEncryptionKeyLength(envOrFail("BLOG_MAILER_ENCRYPTION_KEY", ""))
	xmlFeedUrl                = getEnv("BLOG_MAILER_XML_FEED_URL", "https://blog.mybb.com/feed.xml")
	lastPostFilePath          = getEnv("BLOG_MAILER_LAST_POST_FILE_PATH", "./last_blog_post.txt")
	emailFromName             = getEnv("BLOG_MAILER_FROM_NAME", "MyBB Blog")

	httpClient = &http.Client{
		Timeout: time.Second * 5,
	}

	templates = template.Must(template.New("").Funcs(template.FuncMap{
		"toPlainText": func(target string) string {
			return bluemonday.StrictPolicy().Sanitize(target)
		},
		"stripUnsafeTags": func(target string) string {
			return bluemonday.UGCPolicy().Sanitize(target)
		},
	}).ParseFiles("templates/email.tmpl", "templates/email.html", "templates/signup.html"))
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

func tryGetNewPost() (*newBlogPost, error) {
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

func sendMailNotification() {
	newBlogPost, err := tryGetNewPost()

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

	err = templates.ExecuteTemplate(&plainTextContentBuffer, "email.tmpl", newBlogPost)

	if err != nil {
		log.Printf("[ERROR] unable to create plaintext email content: %s\n", err)

		return
	}

	var htmlContentBuffer bytes.Buffer

	err = templates.ExecuteTemplate(&htmlContentBuffer, "email.html", newBlogPost)

	if err != nil {
		log.Printf("[ERROR] unable to create HTML email content: %s\n", err)

		return
	}

	mg := mailgun.NewMailgun(mailGunDomain, mailGunApiKey, mailGunPublicKey)

	fromAddress := mailGunMailingListAddress

	if len(emailFromName) > 0 {
		fromAddress = fmt.Sprintf("%s <%s>", emailFromName, mailGunMailingListAddress)
	}

	message := mg.NewMessage(
		fromAddress,
		"New MyBB Blog Post: "+newBlogPost.Title,
		plainTextContentBuffer.String(),
		mailGunMailingListAddress)

	message.SetHtml(htmlContentBuffer.String())
	message.AddHeader("List-Unsubscribe", "%unsubscribe_email%")

	resp, id, err := mg.Send(message)
	if err != nil {
		log.Printf("[ERROR] unable to send update email: %s\n", err)
	} else {
		log.Printf("[DEBUG] sent email with id %s and status: %s\n", id, resp)

		lastPostDate := newBlogPost.PublishedAt.Format(time.RFC3339)

		err = ioutil.WriteFile(lastPostFilePath, []byte(lastPostDate), 0644)

		if err != nil {
			log.Printf("[WARN] unable to save last post date '%s': %s\n", lastPostDate, err)
		}
	}
}

func handleWebHook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, secret)
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

func handleSubscribe(writer http.ResponseWriter, request *http.Request) {
	email, _ := getSubscribeEmail(request)
	// TODO: If there's an error, show an error page instead...

	if len(email) > 0 {
		// TODO: Subscribe to mailing list and show success page...
	}

	templates.ExecuteTemplate(writer, "signup.html", nil)
}

func getSubscribeEmail(request *http.Request) (string, error) {
	query := request.URL.Query()

	if keys, ok := query["email"]; !ok || len(keys[0]) < 1 {
		return "", nil
	} else {
		// email is a base64 encoded AES GCM encrypted email address - it is done this way to ensure the request to subscribe was actually confirmed from the confirmation email
		base64Email := keys[0]

		decodedBytes, err := base64.URLEncoding.DecodeString(base64Email)

		if err != nil {
			return "", fmt.Errorf("error base64 decoding email: %s", err)
		}

		decryptedBytes, err := decrypt(decodedBytes)

		if err != nil {
			return "", fmt.Errorf("error decrypting email: %s", err)
		}

		return string(decryptedBytes), nil
	}
}

func encrypt(text []byte) ([]byte, error) {
	createdCipher, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(createdCipher)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, text, nil), nil
}

func decrypt(encrypted []byte) ([]byte, error) {
	c, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, errors.New("encrypted text too short")
	}

	nonce, encrypted := encrypted[:nonceSize], encrypted[nonceSize:]
	return gcm.Open(nil, nonce, encrypted, nil)
}

func main() {
	http.HandleFunc("/webhook", handleWebHook)
	http.HandleFunc("/subscribe", handleSubscribe)

	address := ":" + port

	log.Printf("[DEBUG] starting webhook server: %s", address)

	log.Fatalln(http.ListenAndServe(address, nil))
}
