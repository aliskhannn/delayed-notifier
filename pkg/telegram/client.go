// Package telegram provides a simple client for sending notifications via Telegram.
//
// It allows creating a client with a bot token and sending messages to specified chat IDs.
// Designed to be used as a notifier in the delayed-notifier system.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client represents a Telegram client used to send notifications.
type Client struct {
	token  string       // bot token for authentication
	client *http.Client // HTTP client used to make requests
}

// NewClient creates a new Telegram Client instance with the given bot token.
func NewClient(token string) *Client {
	return &Client{
		token:  token,
		client: &http.Client{},
	}
}

// sendMessageRequest represents the payload for the Telegram sendMessage API.
type sendMessageRequest struct {
	ChatID string `json:"chat_id"` // chat id to send message to
	Text   string `json:"text"`    // message text
}

// Send sends a notification message to the specified Telegram chat ID.
//
// It constructs the request payload, sends an HTTP POST to the Telegram Bot API,
// and returns an error if the request fails or the API responds with a non-200 status.
func (c *Client) Send(to string, msg string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token) // telegram API URL

	reqBody := sendMessageRequest{
		ChatID: to,  // recipient chat id
		Text:   msg, // message text
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error: %s", resp.Status)
	}

	return nil
}
