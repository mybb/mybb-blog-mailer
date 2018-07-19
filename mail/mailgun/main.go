package mailgun

import (
	"github.com/mybb/mybb-blog-mailer/config"
	"github.com/mybb/mybb-blog-mailer/mail"
	"gopkg.in/mailgun/mailgun-go.v1"
)

/// Handler wraps a MailGun API client to make it easy to send emails to perform tasks related to emails.
type Handler struct {
	client mailgun.Mailgun
}

/// NewHandler creates a new MailGun mail handler using the given configuration.
func NewHandler(configuration *config.MailGunConfig) *Handler {
	return &Handler{
		client: mailgun.NewMailgun(
			configuration.Domain,
			configuration.ApiKey,
			configuration.PublicKey,
		),
	}
}

/// CheckValidEmail checks whether the given email address is a valid email address using the MailGun API.
func (h *Handler) CheckValidEmail(emailAddress string) (bool, error) {
	if len(emailAddress) == 0 {
		return false, mail.EmptyEmailAddressError{}
	}

	ev, err := h.client.ValidateEmail(emailAddress)

	if err != nil {
		return false, err
	}

	if !ev.IsValid {
		return false, nil
	}

	return true, nil
}
