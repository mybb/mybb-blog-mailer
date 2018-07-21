package config

import (
	"math"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"fmt"
)

/// MailGun holds configuration for sending email notifications via MailGun.
type MailGunConfig struct {
	/// Domain is the domain name configured with MailGun to send emails from.
	Domain string `toml:"domain"`
	/// ApiKey is the API key provided by MailGun to communicate with their API.
	ApiKey string `toml:"api_key"`
	/// PublicKey is the public key provided by MailGun to communicate with their API.
	PublicKey string `toml:"public_key"`
	/// MailingListAddress is the email address of the mailing list to send email notifications to.
	MailingListAddress string `toml:"mailing_list"`
	/// FromName is the name to show with email notifications sent to the mailing list.
	FromName string `toml:"from_name"`
	/// EmailValidation determines whether to use MailGun's email validation API. This requires a paid MailGun account.
	EmailValidation bool `toml:"email_validation"`
}

/// Config holds application configuration.
type Config struct {
	/// ListenPort is the TCP port to listen for HTTP requests on.
	ListenPort int `toml:"listen_port"`
	/// WebHookSecret is a secret configured with the GitHub webhook to verify requests originate from GitHub.
	WebHookSecret string `toml:"web_hook_secret"`
	/// XmlFeedUrl is the URL to check for blog posts for a successful GitHub pages build.
	XmlFeedUrl string `toml:"xml_feed_url"`
	/// HmacSecret is the secret phrase used when signing an email during email verification to ensure authenticity.
	HmacSecret string `toml:"hmac_secret"`
	/// MailGun is the configuration related to sending email notifications via MailGun.
	MailGun MailGunConfig `toml:"mailgun"`
}

/// InitConfigFromConfigFile reads and parses a config file into a config structure.
func InitConfigFromConfigFile(filePath string) (*Config, error) {
	config := Config{
		ListenPort: 80,
		XmlFeedUrl: "https://blog.mybb.com/feed.xml",
		MailGun: MailGunConfig{
			FromName: "MyBB Blog",
			EmailValidation: false,
		},
	}

	configFileContent, err := ioutil.ReadFile(filePath)

	if err != nil {
		return nil, fmt.Errorf("error reading config file '%s': %s", filePath, err)
	}

	err = toml.Unmarshal(configFileContent, &config)

	if err != nil {
		return nil, fmt.Errorf("error parsing config file '%s': %s", filePath, err)
	}

	err = config.validate()

	if err != nil {
		return nil, err
	}

	return &config, nil
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
