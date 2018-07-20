package config

import (
	"flag"
	"math"
	"net/url"
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
	XmlFeedUrl *url.URL
	/// HmacSecret is the secret phrase used when signing an email during email verification to ensure authenticity.
	HmacSecret string
	/// MailGun is the configuration related to sending email notifications via MailGun.
	MailGun MailGunConfig
}

/// InitConfigFromFlags initialises a configurations structure from command line arguments.
func InitConfigFromFlags() (*Config, error) {
	config := Config{
		MailGun: MailGunConfig{},
	}

	var xmlFeedUrl string

	flag.IntVar(&config.ListenPort, "port", 80, "The TCP port to listen for HTTP requests on")
	flag.StringVar(&config.WebHookSecret, "gh-secret", "",
		"Secret configured with the GitHub webhook to verify requests originate from GitHub")
	flag.StringVar(&xmlFeedUrl, "xml-feed", "https://blog.mybb.com/feed.xml",
		"The URL to check for blog posts for a successful GitHub pages build")
	flag.StringVar(&config.HmacSecret, "hmac-secret", "",
		"The secret phrase used when signing an email during email verification to ensure authenticity")

	flag.StringVar(&config.MailGun.Domain, "mg-domain", "",
		"The domain name configured with MailGun to send emails from")
	flag.StringVar(&config.MailGun.ApiKey, "mg-api-key", "",
		"The API key provided by MailGun to communicate with their API")
	flag.StringVar(&config.MailGun.PublicKey, "mg-public-key", "",
		"The public key provided by MailGun to communicate with their API")
	flag.StringVar(&config.MailGun.MailingListAddress, "mg-mailing-list", "",
		"The email address of the mailing list to send email notifications to")
	flag.StringVar(&config.MailGun.FromName, "mg-from-name", "MyBB Blog",
		"The name to show with email notifications sent to the mailing list")
	flag.BoolVar(&config.MailGun.EmailValidation, "mg-email-validation", false,
		"Whether to use MailGun's email validation API. This requires a paid MailGun account")

	flag.Parse()

	var err error

	config.XmlFeedUrl, err = url.Parse(xmlFeedUrl)
	if err != nil {
		return &config, err
	}

	return &config, config.validate()
}

func (c *Config) validate() error {
	if c.ListenPort < 1 || c.ListenPort > math.MaxUint16 {
		return OutOfRangeError{
			ParameterName: "port",
		}
	}

	if len(c.WebHookSecret) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "gh-secret",
		}
	}

	if len(c.HmacSecret) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "hmac-secret",
		}
	}

	if len(c.MailGun.Domain) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "mg-domain",
		}
	}

	if len(c.MailGun.ApiKey) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "mg-api-key",
		}
	}

	if len(c.MailGun.PublicKey) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "mg-public-key",
		}
	}

	if len(c.MailGun.MailingListAddress) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "mg-mailing-list",
		}
	}

	if len(c.MailGun.FromName) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "mg-from-name",
		}
	}

	return nil
}
