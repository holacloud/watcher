package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/fulldump/biff"
)

func TestSendMessageHttp(t *testing.T) {

	telegramMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		AssertEqual(r.Header.Get("Content-Type"), "application/json")

		payload := map[string]string{}
		json.NewDecoder(r.Body).Decode(&payload)
		AssertEqual(payload["chat_id"], "my-chat")
		AssertEqual(payload["text"], "my-text")
	}))
	defer telegramMock.Close()

	err := sendMessageHttp(http.DefaultClient, telegramMock.URL+"/my-endpoint", "my-chat", "my-text")

	AssertNil(err)
}

func TestSendMessage(t *testing.T) {

	telegramMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, "payload-with-error")
	}))
	defer telegramMock.Close()

	err := sendMessageHttp(http.DefaultClient, telegramMock.URL+"/my-endpoint", "my-chat", "my-text")

	AssertNotNil(err)
	AssertEqual(err.Error(), "unexpected status code '500': payload-with-error")
}

func TestNew(t *testing.T) {
	telegramMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))
	defer telegramMock.Close()

	tel := New(Config{
		BaseURL: telegramMock.URL,
		Bot:     "my-bot",
		Chat:    "my-chat",
	})

	err := tel.SendMessage("Hello world")
	AssertNil(err)
}

func TestRealFire(t *testing.T) {

	t.SkipNow()

	tel := New(Config{
		Bot:  "",
		Chat: "",
	})

	err := tel.SendMessage("Hello world")
	AssertNil(err)
}
