package mail

type EmptyEmailAddressError struct{}

func (e EmptyEmailAddressError) Error() string {
	return "email address is empty"
}
