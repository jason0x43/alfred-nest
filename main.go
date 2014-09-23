package main

import (
	"fmt"
	"github.com/jason0x43/go-alfred"
	"github.com/jason0x43/go-keychain"
	"github.com/jason0x43/go-log"
	"github.com/jason0x43/go-nest"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"
)

type Config struct {
	Nest  string
	Email string
}

type Cache struct {
	Time    time.Time
	Session nest.Session
	Status  nest.Status
}

var cacheFile string
var configFile string
var config Config
var cache Cache
var session nest.Session

const (
	KEYWORD = "nst"
)

// usage: ./alfred-nest {do,tell} keyword query
func main() {
	configFile = path.Join(alfred.DataDir(), "config.json")
	log.Println("Using config file", configFile)

	cacheFile = path.Join(alfred.CacheDir(), "cache.json")
	log.Println("Using cache file", cacheFile)

	err := alfred.LoadJson(configFile, &config)
	if err != nil {
		log.Println("Error loading config:", err)
	}

	err = alfred.LoadJson(cacheFile, &cache)
	if err != nil {
		log.Println("Error loading cache:", err)
	}

	session = cache.Session
	log.Println("Initialized session:", session)

	filters := []alfred.Filter{
		TempCommand{},
		ConfigCommand{},
		SyncCommand{}}
	actions := []alfred.Action{
		SyncCommand{},
		OpenCommand{},
		TempCommand{}}

	alfred.Run(filters, actions)
}

// config ----------------------------------------------------------------------

type ConfigCommand struct{}

func (t ConfigCommand) Keyword() string {
	return "config"
}

func (t ConfigCommand) MenuItem() alfred.Item {
	items, _ := t.Items("", "")
	return items[0]
}

func (t ConfigCommand) Items(prefix, query string) ([]alfred.Item, error) {
	items := []alfred.Item{alfred.Item{
		Title:        "config",
		Arg:          "open " + strconv.Quote(configFile),
		Autocomplete: "config",
		Subtitle:     "Open the nest config"}}

	return items, nil
}

// open ----------------------------------------------------------------------

type OpenCommand struct{}

func (t OpenCommand) Keyword() string {
	return "open"
}

func (t OpenCommand) Do(query string) (string, error) {
	log.Printf("open '%s'\n", query)
	name, err := strconv.Unquote(query)
	if err == nil {
		err = exec.Command("open", name).Run()
	}

	return "", err
}

// temp ----------------------------------------------------------------------

type TempCommand struct{}

func (t TempCommand) Keyword() string {
	return "temp"
}

func (t TempCommand) MenuItem() alfred.Item {
	return alfred.Item{
		Title:        "temp",
		Subtitle:     "Adjust the your home's temperature",
		Autocomplete: "temp "}
}

func (t TempCommand) Items(prefix, query string) ([]alfred.Item, error) {
	if time.Now().Sub(cache.Time).Minutes() > 10.0 {
		log.Println("Refreshing cache...")
		items := []alfred.Item{alfred.Item{
			Title:    "temp",
			Subtitle: "Thinking..."}}
		cmd := exec.Command("./alfred-nest", "do", "sync", KEYWORD, "temp ")
		cmd.Start()

		return items, nil
	}

	var items []alfred.Item

	// if time.Now().Sub(cache.Time).Minutes() > 10.0 {
	// 	log.Println("Refreshing cache...")
	// 	err := refresh()
	// 	if err != nil {
	// 		return items, err
	// 	}
	// }

	log.Printf("temp query: %s", query)

	shared, ok := getShared()
	if !ok {
		return items, fmt.Errorf("No active Nest")
	}

	// add 0.5 to round
	temp := int(nest.ToFahrenheit(shared.CurrentTemp) + 0.5)

	if query != "" {
		newTemp, err := strconv.Atoi(query)
		if err == nil && newTemp != temp {
			newTempC := nest.ToCelsius(float64(newTemp))
			items = append(items, alfred.Item{
				Title: fmt.Sprintf("Set temperature to %d°F", newTemp),
				Arg:   "temp " + strconv.FormatFloat(newTempC, 'f', 3, 64)})
			return items, nil
		}
	}

	subtitle := "Target is "
	if shared.TargetTempType == "range" {
		targetHigh := int(nest.ToFahrenheit(shared.TargetTempHigh) + 0.5)
		targetLow := int(nest.ToFahrenheit(shared.TargetTempLow) + 0.5)
		subtitle += fmt.Sprintf("%d°F to %d°F", targetLow, targetHigh)
	} else {
		target := int(nest.ToFahrenheit(shared.TargetTemp) + 0.5)
		subtitle += strconv.Itoa(target) + "°F"
	}

	items = append(items, alfred.Item{
		Title:    fmt.Sprintf("Current temperature is %d°F", temp),
		Subtitle: subtitle,
		Valid:    alfred.INVALID})

	return items, nil
}

func (t TempCommand) Do(query string) (string, error) {
	shared, ok := getShared()
	if !ok {
		return "", fmt.Errorf("Error accessing Nest -- try syncing")
	}

	newTemp, err := strconv.ParseFloat(query, 64)
	if err != nil {
		return "", err
	}

	var target nest.TargetTemperature

	switch shared.TargetTempType {
	case "heat":
		fallthrough
	case "cool":
		target.TargetTemp = newTemp
	default:
		currentTemp := shared.CurrentTemp

		// range must be 3°F at a minimum; use 4°F
		if newTemp < currentTemp {
			target.TargetTempHigh = newTemp
			target.TargetTempLow = newTemp - 2.222
		} else {
			target.TargetTempLow = newTemp
			target.TargetTempHigh = newTemp + 2.222
		}
	}

	log.Printf("Setting temp to %#v\n", target)
	err = session.UpdateTargetTemp(config.Nest, target)

	temp := int(nest.ToFahrenheit(newTemp) + 0.5)
	return fmt.Sprintf("Set temperature to %d°F", temp), err
}

// sync ----------------------------------------------------------------------

type SyncCommand struct{}

func (t SyncCommand) Keyword() string {
	return "sync"
}

func (t SyncCommand) MenuItem() alfred.Item {
	return alfred.Item{
		Title:        "sync",
		Subtitle:     "Synchronize with Nest.com",
		Autocomplete: "sync"}
}

func (t SyncCommand) Items(prefix, query string) ([]alfred.Item, error) {
	items := []alfred.Item{alfred.Item{
		Title:    "sync",
		Subtitle: "Thinking..."}}
	cmd := exec.Command("./alfred-nest", "do", "sync", KEYWORD+" ")
	err := cmd.Start()
	return items, err
}

func (t SyncCommand) Do(query string) (string, error) {
	syncFile := path.Join(alfred.CacheDir(), "sync.lock")
	f, err := os.OpenFile(syncFile, os.O_CREATE|os.O_EXCL, 0600)
	if err == nil {
		err = refresh()
		f.Close()
		os.Remove(syncFile)

		if err != nil {
			alfred.ShowMessage("Error", fmt.Sprintf("There was an error syncing with Nest:\n\n%s", err))
		} else {
			alfred.RunScript(`tell application "Alfred 2" to run trigger "message" in workflow "com.jason0x43.alfred.nest" with argument "Synchronized!"`)
			alfred.RunScript(fmt.Sprintf(`tell application "Alfred 2" to search "%s"`, query))
		}
	}
	return "", err
}

// support -------------------------------------------------------------------

func getShared() (nest.Shared, bool) {
	for id, s := range cache.Status.Shared {
		if id == config.Nest {
			return s, true
		}
	}
	return nest.Shared{}, false
}

func getDevice() (nest.Device, bool) {
	for id, s := range cache.Status.Device {
		if id == config.Nest {
			return s, true
		}
	}
	return nest.Device{}, false
}

func refresh() error {
	if !session.IsValid() {
		log.Println("Re-establishing session")
		username := "j.cheatham@gmail.com"
		password, err := keychain.GetPassword(KEYWORD, "jc-nest")
		if err != nil {
			return err
		}

		session, err = nest.NewSession(username, password)
		if err != nil {
			return err
		}

		cache.Session = session
		alfred.SaveJson(cacheFile, &cache)
	}

	log.Println("Getting status...")
	status, err := session.GetStatus()
	if err != nil {
		log.Println("Errror getting status:", err)
		return err
	}

	cache.Status = status
	cache.Time = time.Now()
	alfred.SaveJson(cacheFile, &cache)
	return nil
}
