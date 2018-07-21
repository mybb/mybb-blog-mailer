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
	"io/ioutil"
	"encoding/base64"

	"github.com/gorilla/mux"
	"github.com/gorilla/csrf"

	"github.com/mybb/mybb-blog-mailer/config"
	"github.com/mybb/mybb-blog-mailer/mail/mailgun"
	"github.com/mybb/mybb-blog-mailer/templating"
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
		"Path to store the session key")
	lastPostDateFilePath := flag.String("last_post_path", "./last_post_date",
		"Path to store the date of the last blog post that was sent to subscribers")

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

	templates, err := templating.FindAndParseTemplates("./templates", templating.BuildDefaultFunctionMap())

	if err != nil {
		log.Fatalf("[ERROR] reading templates: %s\n", err)
	}

	subscriptionService := NewSubscriptionService(mailHandler, templates, configuration.HmacSecret,
		sessionKey)
	webHookService := NewWebHookService(mailHandler, templates, configuration.WebHookSecret, configuration.XmlFeedUrl,
		*lastPostDateFilePath)

	router := newRouter(subscriptionService, webHookService)

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
func newRouter(subscriptionService *SubscriptionService, whService *WebHookService) *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/", subscriptionService.Index).Methods("GET").Name("index")
	router.HandleFunc("/signup", subscriptionService.SignUp).Methods("POST").Name("sign_up")
	router.HandleFunc("/confirm", subscriptionService.ConfirmSignUp).Methods("GET").Name(
		"confirm_signup")

	router.HandleFunc("/webhook", whService.Index).Methods("POST").Name("webhook")

	return router
}

/// bindMiddleware wraps a HTTP handler with a stack of middleware.
func bindMiddleware(handler http.Handler, csrfkey []byte) http.Handler {
	var secureOption csrf.Option
	if os.Getenv("DEBUG") == "1" {
		secureOption = csrf.Secure(false)
	} else {
		secureOption = csrf.Secure(true)
	}

	csrfMiddleware := csrf.Protect(csrfkey, secureOption)

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