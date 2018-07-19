package mail

type Handler interface {
	/// CheckValidEmail checks whether the given email address is a valid email address using the MailGun API.
	CheckValidEmail(emailAddress string) (bool, error)
}
