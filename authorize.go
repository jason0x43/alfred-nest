package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"os/exec"

	"github.com/jason0x43/go-alfred"
)

// AuthorizeCommand authorizes a user
type AuthorizeCommand struct{}

// About returns information about a command
func (c AuthorizeCommand) About() alfred.CommandDef {
	return alfred.CommandDef{
		Keyword:     "authorize",
		Description: "Authorize this workflow to access your Nest",
		IsEnabled:   config.AccessToken == "",
		Arg: &alfred.ItemArg{
			Keyword: "authorize",
			Mode:    alfred.ModeDo,
		},
	}
}

// Do runs the command
func (c AuthorizeCommand) Do(query string) (out string, err error) {
	if err = StartAuthServer(); err != nil {
		return
	}

	randBytes := make([]byte, 32)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}
	randString := base64.URLEncoding.EncodeToString(randBytes)

	params := url.Values{}
	params.Add("client_id", clientID)
	params.Add("state", randString)

	oauthURL := "https://home.nest.com/login/oauth2?" + params.Encode()
	return "", exec.Command("open", oauthURL).Run()
}
