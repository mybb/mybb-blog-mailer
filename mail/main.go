package mail

import (
	"github.com/goware/emailx"
)

type Handler interface {
	/// CheckValidEmail checks whether the given email address is a valid email address using the MailGun API.
	CheckValidEmail(emailAddress string) (bool, error)
	/// SendSubscriptionConfirmationEmail sends an email to the given address to confirm their subscription to the mailing list.
	SendSubscriptionConfirmationEmail(emailAddress string, textContent, htmlContent string) error
	/// Subscribe the given email address to the mailing list with the given name.
	SubscribeEmailToMailingList(emailAddress, name string) error
}

/// ValidateEmailAddress checks whether an email address is valid.
func ValidateEmailAddress(emailAddress string) (bool, error) {
	err := emailx.Validate(emailAddress)

	return err == nil, err
}