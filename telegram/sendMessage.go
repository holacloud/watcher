package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Config struct {
	BaseURL string `usage:"https://api.telegram.org"`
	Bot     string `usage:"Telegram Bot ID"`
	Chat    string `usage:"Telegram Chat ID"`
}

type Telegram struct {
	Config     Config
	HttpClient *http.Client
}

func New(c Config) *Telegram {

	if c.BaseURL == "" {
		c.BaseURL = "https://api.telegram.org"
	}

	return &Telegram{
		c,
		http.DefaultClient,
	}
}

func (t *Telegram) SendMessage(text string) error {

	go func() {
		err := t.SendMessageSync(text)
		if err != nil {
			fmt.Println("TELEGRAM ERROR:", err.Error())
		}
	}()
	return nil
}

func (t *Telegram) SendMessageSync(text string) error {
	fmt.Println("TELEGRAM:", text)
	if t.Config.Bot == "" || t.Config.Chat == "" {
		return nil
	}
	u := fmt.Sprintf("%s/%s/sendMessage", t.Config.BaseURL, t.Config.Bot)
	return sendMessageHttp(t.HttpClient, u, t.Config.Chat, text)
}

func sendMessageHttp(c *http.Client, endpoint, chat, text string) error {

	payload, err := json.Marshal(map[string]interface{}{
		"chat_id": chat,
		"text":    text,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code '%d': %s", resp.StatusCode, string(b))
	}

	io.Copy(io.Discard, resp.Body)
	return nil
}
