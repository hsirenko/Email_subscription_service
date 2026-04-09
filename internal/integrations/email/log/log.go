package log

import (
	"log"
)

type Sender struct{}

func (s Sender) SendConfirm(toEmail string, confirmURL string, unsubscribeURL string) error {
	log.Printf("CONFIRM to=%s url=%s", toEmail, confirmURL)
	log.Printf("UNSUBSCRIBE to=%s url=%s", toEmail, unsubscribeURL)
	return nil
}

func (s Sender) SendRelease(toEmail string, repoFullName string, tag string, releaseURL string, unsubscribeURL string) error {
	log.Printf("RELEASE to=%s repo=%s tag=%s url=%s", toEmail, repoFullName, tag, releaseURL)
	log.Printf("UNSUBSCRIBE to=%s url=%s", toEmail, unsubscribeURL)
	return nil
}
