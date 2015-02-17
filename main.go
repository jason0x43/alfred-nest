package main

import (
	"log"
	"path"
	"time"

	"github.com/jason0x43/go-alfred"
)

type Config struct {
	NestId       string
	AccessToken  string
	AccessExpiry time.Time
	Scale        TempScale
}

type Cache struct {
	Time    time.Time
	AllData AllData
}

const (
	ClientId     = "359f0dd0-8935-4390-9f10-863a5b7ec606"
	CallbackPath = "/oauth/callback"
	CallbackPort = "2222"
	RedirectUri  = "http://localhost:" + CallbackPort + CallbackPath
)

//go:generate go build support/oauthgen.go
//go:generate ./oauthgen credentials.go

var ClientSecret string
var cacheFile string
var configFile string
var config Config
var cache Cache

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
