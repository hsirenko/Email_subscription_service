package email

// Sender sends transactional email for subscription confirm and release alerts.
type Sender interface {
	SendConfirm(toEmail string, confirmURL string, unsubscribeURL string) error
	SendRelease(toEmail string, repoFullName string, tag string, releaseURL string, unsubscribeURL string) error
}
