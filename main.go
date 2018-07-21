package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"crypto/rand"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/gorilla/csrf"

	"github.com/mybb/mybb-blog-mailer/config"
	"github.com/mybb/mybb-blog-mailer/mail/mailgun"
	"io/ioutil"
	"encoding/base64"
)

// init sets basic runtime settings for the application.
func init() {
	log.SetFlags(log.LstdFlags)
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	configFilePath := flag.String("config", "./.env",
		"Optional path to a .env configuration file")
	storedCsrfKeyFilePath := flag.String("csrf_key_path", "./.csrf_key",
		"Path to store the CSRF key")
	storedSessionKeyPath := flag.String("session_key_path", "./.session_key",
		"Path to store the sesison key")

	flag.Parse()

	configuration, err := config.InitFromEnvironment(*configFilePath)

	if err != nil {
		log.Printf("[ERROR] initialising configuration: %s\n", err)
		flag.Usage()
		os.Exit(1)
	}

	sessionKey, err := readOrGenerateKey(*storedSessionKeyPath)

	if err != nil {
		log.Fatalf("[ERROR] reading or generating session key: %s\n", err)
	}

	mailHandler := mailgun.NewHandler(&configuration.MailGun)

	subscriptionService, err := NewSubscriptionService(mailHandler, configuration.HmacSecret,
		sessionKey)

	if err != nil {
		log.Fatalf("[ERROR] initialising subscription service: %s\n", err)
	}

	router := newRouter(subscriptionService)

	csrfKey, err := readOrGenerateKey(*storedCsrfKeyFilePath)

	if err != nil {
		log.Fatalf("[ERROR] reading or generating CSRF key: %s\n", err)
	}

	log.Printf("[DEBUG] starting HTTP server on :%d\n", configuration.ListenPort)

	err = http.ListenAndServe(":"+strconv.Itoa(configuration.ListenPort), bindMiddleware(router, csrfKey))

	if err != nil {
		log.Fatalf("[ERROR] running HTTP server: %s\n", err)
	}
}

/// newRouter creates and configures a HTTP router to dispatch requests to handlers.
func newRouter(subscriptionService *SubscriptionService) *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/", subscriptionService.Index).Methods("GET").Name("index")
	router.HandleFunc("/signup", subscriptionService.SignUp).Methods("POST").Name("sign_up")
	router.HandleFunc("/confirm", subscriptionService.ConfirmSignUp).Methods("GET").Name(
		"confirm_signup")

	return router
}

/// bindMiddleware wraps a HTTP handler with a stack of middleware.
func bindMiddleware(handler http.Handler, csrfkey []byte) http.Handler {
	csrfMiddleware := csrf.Protect(csrfkey, csrf.Secure(false)) // TODO: Remove secure(false) for production...

	return csrfMiddleware(handler)
}

/// readOrGenerateKey reads a 32 byte key from a base64 encoded file, or generates a new key f the file doesn't exist or is empty.
func readOrGenerateKey(keyFilePath string) ([]byte, error) {
	if len(keyFilePath) > 0 {
		if storedKey, err := ioutil.ReadFile(keyFilePath); err == nil {
			if base64Decoded, err := base64.StdEncoding.DecodeString(string(storedKey));
			err == nil && len(base64Decoded) == 32 {
				return base64Decoded, nil
			}
		}
	}

	b := make([]byte, 32)

	_, err := rand.Read(b)

	if err != nil {
		return nil, fmt.Errorf("error reading random bytes for CSRF auth key: %s", err)
	}

	if len(keyFilePath) > 0 {
		b64 := base64.StdEncoding.EncodeToString(b)
		ioutil.WriteFile(keyFilePath, []byte(b64), 0644)
	}

	return b, nil
}

//package main
//
//type BlogMailerService interface {
//	Subscribe(email string) error
//
//}
////
////type newBlogPost struct {
////	Title       string
////	Summary     string
////	Url         string
////	PublishedAt time.Time
////	Author      string
////}
//
//func main() {
//	//listenPort := flag.Int("port", 80, "The TCP port to listen for HTTP requests on")
//	//githubHookSecret := flag.String("gh-secret", "", "Secret used to confirm WebHook requests are sent by GitHub")
//	//mailGunDomain := flag.String("mg-domain", "", "The domain name to send emails from, must be configured within MailGun")
//	//mailGunApiKey := flag.String("mg-api-key", "", "The API key used to communicate with MailGun")
//	//mailGunPublicKey := flag.String("mg-public-key", "", "The public key used to communicate with MailGun")
//	//mailGunMailingListAddress := flag.String("mg-mailing-list", "", "The email address of the mailing list to send email notifications to")
//	//mailGunEmailFromName := flag.String("mg-from-name", "MyBB Blog", "The name to use as the sender name when sending email notifications")
//	//xmlFeedUrl := flag.String("xml-feed", "https://blog.mybb.com/feed.xml", "The URL to the XML feed to read in order to find new posts")
//	//hmacSecret := flag.String("hmac-secret", "", "Secret used to sign email addresses to verify email notification sign-ups")
//	//
//	//
//	//
//	//http.HandleFunc("/webhook", handleWebHook)
//	//http.HandleFunc("/subscribe", handleSubscribe)
//	//
//	//address := ":" + port
//	//
//	//log.Printf("[DEBUG] starting webhook server: %s", address)
//	//
//	//log.Fatalln(http.ListenAndServe(address, nil))
//}
////
////func getLastPostDate() (*time.Time, error) {
////	fileContent, err := ioutil.ReadFile(lastPostFilePath)
////
////	if err != nil {
////		return nil, err
////	}
////
////	if len(fileContent) == 0 {
////		return nil, nil
////	}
////
////	parsedTime, err := time.Parse(time.RFC3339, string(fileContent))
////
////	if err != nil {
////		return nil, err
////	}
////
////	return &parsedTime, nil
////}
////
////func tryGetNewPost() (*newBlogPost, error) {
////	resp, err := httpClient.Get(xmlFeedUrl)
////
////	if err != nil {
////		return nil, err
////	}
////
////	defer resp.Body.Close()
////
////	feedParser := gofeed.NewParser()
////
////	feed, err := feedParser.Parse(resp.Body)
////
////	if err != nil {
////		return nil, err
////	}
////
////	if len(feed.Items) == 0 {
////		return nil, nil
////	}
////
////	lastPostDate, err := getLastPostDate()
////
////	if err != nil && !os.IsNotExist(err) {
////		return nil, err
////	}
////
////	// We only check the most recent post
////
////	mostRecentPost := feed.Items[0]
////
////	if lastPostDate != nil && (mostRecentPost.PublishedParsed == nil || !mostRecentPost.PublishedParsed.After(*lastPostDate)) {
////		return nil, nil
////	}
////
////	author := ""
////
////	if mostRecentPost.Author != nil {
////		author = mostRecentPost.Author.Name
////	}
////
////	return &newBlogPost{
////		Title:       mostRecentPost.Title,
////		Summary:     mostRecentPost.Description,
////		Url:         mostRecentPost.Link,
////		PublishedAt: *mostRecentPost.PublishedParsed,
////		Author:      author,
////	}, nil
////}
////
////func sendMailNotification() {
////	newBlogPost, err := tryGetNewPost()
////
////	if err != nil {
////		log.Printf("[ERROR] unable to get new blog post: %s\n", err)
////
////		return
////	}
////
////	if newBlogPost == nil {
////		log.Println("[DEBUG] no new blog post found")
////
////		return
////	} else {
////		log.Printf("[DEBUG] found new blog post: %+v\n", *newBlogPost)
////	}
////
////	var plainTextContentBuffer bytes.Buffer
////
////	err = templates.ExecuteTemplate(&plainTextContentBuffer, "blog_post_notification.txt", newBlogPost)
////
////	if err != nil {
////		log.Printf("[ERROR] unable to create plaintext email content: %s\n", err)
////
////		return
////	}
////
////	var htmlContentBuffer bytes.Buffer
////
////	err = templates.ExecuteTemplate(&htmlContentBuffer, "blog_post_notification.html", newBlogPost)
////
////	if err != nil {
////		log.Printf("[ERROR] unable to create HTML email content: %s\n", err)
////
////		return
////	}
////
////	mg := mailgun.NewMailgun(mailGunDomain, mailGunApiKey, mailGunPublicKey)
////
////	fromAddress := mailGunMailingListAddress
////
////	if len(emailFromName) > 0 {
////		fromAddress = fmt.Sprintf("%s <%s>", emailFromName, mailGunMailingListAddress)
////	}
////
////	message := mg.NewMessage(
////		fromAddress,
////		"New MyBB Blog Post: "+newBlogPost.Title,
////		plainTextContentBuffer.String(),
////		mailGunMailingListAddress)
////
////	message.SetHtml(htmlContentBuffer.String())
////	message.AddHeader("List-Unsubscribe", "%unsubscribe_email%")
////
////	resp, id, err := mg.Send(message)
////	if err != nil {
////		log.Printf("[ERROR] unable to send update email: %s\n", err)
////	} else {
////		log.Printf("[DEBUG] sent email with id %s and status: %s\n", id, resp)
////
////		lastPostDate := newBlogPost.PublishedAt.Format(time.RFC3339)
////
////		err = ioutil.WriteFile(lastPostFilePath, []byte(lastPostDate), 0644)
////
////		if err != nil {
////			log.Printf("[WARN] unable to save last post date '%s': %s\n", lastPostDate, err)
////		}
////	}
////}
////
////func handleWebHook(w http.ResponseWriter, r *http.Request) {
////	payload, err := github.ValidatePayload(r, secret)
////	if err != nil {
////		errorMessage := fmt.Sprintf("error validating request body: %s", err)
////
////		log.Printf("[ERROR] " + errorMessage + "\n")
////
////		http.Error(w, errorMessage, http.StatusBadRequest)
////		return
////	}
////
////	defer r.Body.Close()
////
////	event, err := github.ParseWebHook(github.WebHookType(r), payload)
////	if err != nil {
////		errorMessage := fmt.Sprintf("could not parse webhook: %s", err)
////
////		log.Printf("[ERROR] " + errorMessage + "\n")
////
////		http.Error(w, errorMessage, http.StatusBadRequest)
////		return
////	}
////
////	switch e := event.(type) {
////	case *github.PingEvent:
////		log.Println("[DEBUG] received ping event")
////	case *github.PageBuildEvent:
////		switch buildStatus := e.Build.GetStatus(); buildStatus {
////		case "built":
////			log.Println("[DEBUG] received successful page build event, reading feed to send emails")
////
////			// Build was successful, so get the newest post and send email via MailGun
////			sendMailNotification()
////		case "queued":
////			log.Println("[DEBUG] rceived page build event with queued status")
////		case "building":
////			log.Println("[DEBUG] rceived page build event with building status")
////		case "errored":
////			buildError := e.Build.GetError()
////			buildErrorMessage := ""
////
////			if buildError != nil {
////				buildErrorMessage = buildError.GetMessage()
////			}
////
////			if len(buildErrorMessage) > 0 {
////				log.Printf("[WARN] received page build event with error message: %s\n")
////			} else {
////				log.Println("[WARN] received page build event with error status but no error message")
////			}
////		default:
////			log.Printf("[WARN] received page build event with unknown build status: %s\n", buildStatus)
////		}
////	default:
////		warningMessage := fmt.Sprintf("unknown event type: %s", github.WebHookType(r))
////
////		log.Printf("[WARN] " + warningMessage + "\n")
////
////		http.Error(w, warningMessage, http.StatusNotImplemented)
////		return
////	}
////}
