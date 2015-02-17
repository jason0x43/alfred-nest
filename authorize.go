package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"os"
	"os/exec"

	"github.com/jason0x43/go-alfred"
)

// authorize ---------------------------------------------

type AuthorizeCommand struct{}

func (c AuthorizeCommand) Keyword() string {
	return "authorize"
}

func (c AuthorizeCommand) IsEnabled() bool {
	return config.AccessToken == ""
}

func (c AuthorizeCommand) MenuItem() alfred.Item {
	return alfred.Item{
		Title:        c.Keyword(),
		Autocomplete: c.Keyword(),
		Arg:          "authorize",
		SubtitleAll:  "Authorize this workflow to access your Nest",
	}
}

func (c AuthorizeCommand) Items(prefix, query string) ([]alfred.Item, error) {
	item := c.MenuItem()
	item.Arg = "authorize"
	return []alfred.Item{item}, nil
}

func (c AuthorizeCommand) Do(query string) (string, error) {
	if err := exec.Command(os.Args[0], "do", "serve").Start(); err != nil {
		return "", err
	}

	randBytes := make([]byte, 32)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}
	randString := base64.URLEncoding.EncodeToString(randBytes)

	params := url.Values{}
	params.Add("client_id", ClientId)
	params.Add("state", randString)

	oauthUrl := "https://home.nest.com/login/oauth2?" + params.Encode()
	return "", exec.Command("open", oauthUrl).Run()
}

// auth server -------------------------------------------

type AuthServerCommand struct{}

func (c AuthServerCommand) Keyword() string {
	return "serve"
}

func (c AuthServerCommand) IsEnabled() bool {
	return true
}

func (c AuthServerCommand) Do(query string) (string, error) {
	err := StartAuthServer()
	return "", err
}
