package email

import (
	"fmt"
	"log"
)

type Service interface {
	SendInvite(toEmail, noteName, invitedBy string) error
}

type mockService struct{}

func NewMockService() Service {
	return &mockService{}
}

func (s *mockService) SendInvite(toEmail, noteName, invitedBy string) error {
	log.Printf("MOCK EMAIL: Sending invite to %s for note '%s' by user %s", toEmail, noteName, invitedBy)
	// In a real app, you'd use SendGrid, AWS SES, or an SMTP server.
	fmt.Printf("Invite email sent to %s\n", toEmail)
	return nil
}
