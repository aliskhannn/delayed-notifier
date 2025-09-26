// Package email provides a simple SMTP client for sending notification emails.
//
// It allows creating a client with SMTP configuration and sending plain text messages
// to specified recipients. Designed to be used as a notifier in the delayed-notifier system.
package email

import (
	"gopkg.in/mail.v2"
)

// Client represents an email client used to send notifications via SMTP.
type Client struct {
	smtpHost string // smtp server host
	smtpPort int    // smtp server port
	username string // smtp username
	password string // smtp password
	from     string // sender email address
}

// NewClient creates a new Client instance with the given SMTP configuration.
func NewClient(smtpHost string, smtpPort int, username, password, from string) *Client {
	return &Client{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
		from:     from,
	}
}

// Send sends an email notification to the specified recipient with the given message.
//
// It constructs the email message, sets headers, and uses the SMTP dialer to send it.
func (c *Client) Send(to string, msg string) error {
	message := mail.NewMessage()

	message.SetHeader("From", c.from)
	message.SetHeader("To", to)
	message.SetHeader("Subject", "Notification")

	message.SetBody("text/plain", msg)

	dialer := mail.NewDialer(c.smtpHost, c.smtpPort, c.username, c.password)

	return dialer.DialAndSend(message)
}
