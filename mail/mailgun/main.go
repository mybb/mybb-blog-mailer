package mailgun

import (
	"log"

	"gopkg.in/mailgun/mailgun-go.v1"

	"github.com/mybb/mybb-blog-mailer/config"
	"github.com/mybb/mybb-blog-mailer/mail"
)

/// Handler wraps a MailGun API client to make it easy to send emails to perform tasks related to emails.
type Handler struct {
	client mailgun.Mailgun
	useEmailValidation bool
	mailingListAddress string
}

/// NewHandler creates a new MailGun mail handler using the given configuration.
func NewHandler(configuration *config.MailGunConfig) *Handler {
	return &Handler{
		client: mailgun.NewMailgun(
			configuration.Domain,
			configuration.ApiKey,
			configuration.PublicKey,
		),
		useEmailValidation: configuration.EmailValidation,
		mailingListAddress: configuration.MailingListAddress,
	}
}

/// CheckValidEmail checks whether the given email address is a valid email address using the MailGun API.
func (h *Handler) CheckValidEmail(emailAddress string) (bool, error) {
	if len(emailAddress) == 0 {
		return false, mail.EmptyEmailAddressError{}
	}

	if h.useEmailValidation {
		return h.validateEmailUsingApi(emailAddress)
	}

	return mail.ValidateEmailAddress(emailAddress)
}

func (h *Handler) validateEmailUsingApi(emailAddress string) (bool, error) {
	ev, err := h.client.ValidateEmail(emailAddress)

	if err != nil {
		return false, err
	}

	if !ev.IsValid {
		return false, nil
	}

	return true, nil
}

/// SendSubscriptionConfirmationEmail sends an email to the given address to confirm their subscription to the mailing list.
func (h *Handler) SendSubscriptionConfirmationEmail(emailAddress string, textContent, htmlContent string) error {
	message := h.client.NewMessage(h.mailingListAddress, "Confirm Subscription", textContent, emailAddress)

	message.SetHtml(htmlContent)
	message.AddHeader("List-Unsubscribe", "%unsubscribe_email%")

	resp, id, err := h.client.Send(message)
	if err != nil {
		return err
	}

	log.Printf("Sent email confirmation to %s with id %s and status: %s\n", emailAddress, id, resp)

	return nil
}

/// Subscribe the given email address to the mailing list with the given name.
func (h *Handler) SubscribeEmailToMailingList(emailAddress, name string) error {
	member := mailgun.Member{
		Address: emailAddress,
		Name: name,
	}

	return h.client.CreateMember(true, h.mailingListAddress, member)
}