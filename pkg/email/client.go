package email

import (
	"gopkg.in/mail.v2"
)

type Client struct {
	smtpHost string
	smtpPort int
	username string
	password string
	from     string
}

func NewClient(smtpHost string, smtpPort int, username, password, from string) *Client {
	return &Client{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
		from:     from,
	}
}

func (c *Client) Send(to string, msg string) error {
	message := mail.NewMessage()

	message.SetHeader("From", c.from)
	message.SetHeader("To", to)
	message.SetHeader("Subject", "Notification")

	message.SetBody("text/plain", msg)

	dialer := mail.NewDialer(c.smtpHost, c.smtpPort, c.username, c.password)

	return dialer.DialAndSend(message)
}
