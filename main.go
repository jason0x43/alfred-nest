package main

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/jason0x43/go-alfred"
)

var dlog = log.New(os.Stderr, "[nest] ", log.LstdFlags)

const (
	clientID     = "359f0dd0-8935-4390-9f10-863a5b7ec606"
	callbackPath = "/oauth/callback"
	callbackPort = "2222"
	redirectURI  = "http://localhost:" + callbackPort + callbackPath
)

//go:generate go build support/oauthgen.go
//go:generate ./oauthgen credentials.go

var clientSecret string
var cacheFile string
var configFile string
var config struct {
	NestID       string
	AccessToken  string
	AccessExpiry time.Time
	Scale        TempScale
}
var cache struct {
	Time    time.Time
	AllData AllData
}

// usage: ./alfred-nest {do,tell} "keyword query"
func main() {
	workflow, err := alfred.OpenWorkflow(".", true)
	if err != nil {
		log.Fatalln("Error opening workflow:", err)
	}

	configFile = path.Join(workflow.DataDir(), "config.json")
	log.Println("Using config file", configFile)
	err = alfred.LoadJson(configFile, &config)
	if err != nil {
		log.Println("Error loading config:", err)
	}

	if config.Scale == "" {
		config.Scale = ScaleF
		if err = alfred.SaveJson(configFile, &config); err != nil {
			log.Println("Error updating config:", err)
		}
	}

	cacheFile = path.Join(workflow.CacheDir(), "cache.json")
	log.Println("Using cache file", cacheFile)
	if err = alfred.LoadJson(cacheFile, &cache); err != nil {
		log.Println("Error loading cache:", err)
	}

	commands := []alfred.Command{
		StatusCommand{},
		TempCommand{},
		ModeCommand{},
		PresenceCommand{},
		RefreshCommand{},
		DevicesCommand{},
		ConfigCommand{},
		AuthorizeCommand{},
		AuthServerCommand{},
	}

	workflow.Run(commands)
}
