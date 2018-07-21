package config

import (
	"math"
	"fmt"
	"os"
	
	"github.com/joho/godotenv"

	"github.com/mybb/mybb-blog-mailer/helpers"
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

	config := &Config{
		ListenPort: helpers.GetIntEnv("PORT", 8080),
		WebHookSecret: os.Getenv("WEB_HOOK_SECRET"),
		XmlFeedUrl: helpers.GetEnv("XML_FEED_URL", "https://blog.mybb.com/feed.xml"),
		HmacSecret: os.Getenv("HMAC_SECRET"),
		MailGun: MailGunConfig{
			Domain: os.Getenv("MAILGUN_DOMAIN"),
			ApiKey: os.Getenv("MAILGUN_API_KEY"),
			PublicKey: os.Getenv("MAILGUN_PUBLIC_KEY"),
			MailingListAddress: os.Getenv("MAILING_LIST_ADDRESS"),
			FromName: helpers.GetEnv("EMAIL_FROM_NAME", "MyBB Blog"),
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
			ParameterName: "PORT",
		}
	}

	if len(c.WebHookSecret) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "WEB_HOOK_SECRET",
		}
	}

	if len(c.XmlFeedUrl) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "XML_FEED_URL",
		}
	}

	if len(c.HmacSecret) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "HMAC_SECRET",
		}
	}

	if len(c.MailGun.Domain) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "MAILGUN_DOMAIN",
		}
	}

	if len(c.MailGun.ApiKey) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "MAILGUN_API_KEY",
		}
	}

	if len(c.MailGun.PublicKey) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "MAILGUN_PUBLIC_KEY",
		}
	}

	if len(c.MailGun.MailingListAddress) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "MAILING_LIST_ADDRESS",
		}
	}

	if len(c.MailGun.FromName) == 0 {
		return RequiredConfigMissingError{
			ParameterName: "EMAIL_FROM_NAME",
		}
	}

	return nil
}
