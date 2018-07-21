package config

import (
	"math"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

/// MailGun holds configuration for sending email notifications via MailGun.
type MailGunConfig struct {
	/// Domain is the domain name configured with MailGun to send emails from.
	Domain string
	/// ApiKey is the API key provided by MailGun to communicate with their API.
	ApiKey string
	/// PublicKey is the public key provided by MailGun to communicate with their API.
	PublicKey string
	/// MailingListAddress is the email address of the mailing list to send email notifications to.
	MailingListAddress string
	/// FromName is the name to show with email notifications sent to the mailing list.
	FromName string
	/// EmailValidation determines whether to use MailGun's email validation API. This requires a paid MailGun account.
	EmailValidation bool
}

/// Config holds application configuration.
type Config struct {
	/// ListenPort is the TCP port to listen for HTTP requests on.
	ListenPort int
	/// WebHookSecret is a secret configured with the GitHub webhook to verify requests originate from GitHub.
	WebHookSecret string
	/// XmlFeedUrl is the URL to check for blog posts for a successful GitHub pages build.
	XmlFeedUrl string
	/// HmacSecret is the secret phrase used when signing an email during email verification to ensure authenticity.
	HmacSecret string
	/// MailGun is the configuration related to sending email notifications via MailGun.
	MailGun MailGunConfig
}

func InitFromEnvironment(dotEnvFile string) (*Config, error) {
	if len(dotEnvFile) > 0 {
		err := godotenv.Load(dotEnvFile)

		if err != nil {
			return nil, fmt.Errorf("error loading .env file: %s", err)
		}
	}

	var listenPort int
	setListenPortEnv := false
	listenPortStr, ok := os.LookupEnv("LISTEN_PORT")

	if !ok || len(listenPortStr) == 0 {
		listenPort = 8080
		setListenPortEnv = true
	} else {
		if parsedListenPort, err := strconv.Atoi(listenPortStr); err != nil {
			listenPort = 8080
			setListenPortEnv = true
		} else {
			listenPort = parsedListenPort
		}
	}

	if setListenPortEnv {
		os.Setenv("PORT", strconv.Itoa(listenPort))
	}

	xmlFeedUrl, ok := os.LookupEnv("XML_FEED_URL")
	if !ok || len(xmlFeedUrl) == 0 {
		xmlFeedUrl = "https://blog.mybb.com/feed.xml"

		os.Setenv("XML_FEED_URL", xmlFeedUrl)
	}

	fromName, ok := os.LookupEnv("EMAIL_FROM_NAME")
	if !ok || len(fromName) == 0 {
		fromName = "MyBB Blog"

		os.Setenv("EMAIL_FROM_NAME", fromName)
	}

	config := &Config{
		ListenPort: listenPort,
		WebHookSecret: os.Getenv("WEB_HOOK_SECRET"),
		XmlFeedUrl: xmlFeedUrl,
		HmacSecret: os.Getenv("HMAC_SECRET"),
		MailGun: MailGunConfig{
			Domain: os.Getenv("MAILGUN_DOMAIN"),
			ApiKey: os.Getenv("MAILGUN_API_KEY"),
			PublicKey: os.Getenv("MAILGUN_PUBLIC_KEY"),
			MailingListAddress: os.Getenv("MAILING_LIST_ADDRESS"),
			FromName: fromName,
			EmailValidation: os.Getenv("MAILGUN_EMAIL_VALIDATION") == "1",
		},
	}

	err := config.validate()

	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.ListenPort < 1 || c.ListenPort > math.MaxUint16 {
		return OutOfRangeError{
			ParameterName: "listen_port",
		}
	}

	if len(c.WebHookSecret) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "web_hook_secret",
		}
	}

	if len(c.XmlFeedUrl) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "xml_feed_url",
		}
	}

	if len(c.HmacSecret) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "hmac_secret",
		}
	}

	if len(c.MailGun.Domain) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "domain",
		}
	}

	if len(c.MailGun.ApiKey) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "api_key",
		}
	}

	if len(c.MailGun.PublicKey) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "public_key",
		}
	}

	if len(c.MailGun.MailingListAddress) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "mailing_list",
		}
	}

	if len(c.MailGun.FromName) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "from_name",
		}
	}

	return nil
}
