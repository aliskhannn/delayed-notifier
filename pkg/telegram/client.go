package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	token  string
	client *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:  token,
		client: &http.Client{},
	}
}

type sendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func (c *Client) Send(chatID int64, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)

	reqBody := sendMessageRequest{
		ChatID: chatID,
		Text:   text,
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
