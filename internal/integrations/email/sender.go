package email

type Sender interface {
	SendConfirm(toEmail string, confirmURL string, unsubscribeURL string) error
	SendRelease(toEmail string, repoFullName string, tag string, releaseURL string, unsubscribeURL string) error
}