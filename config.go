package main

import (
	"encoding/json"
	"strings"

	"github.com/jason0x43/go-alfred"
)

// ConfigCommand configs the workflow
type ConfigCommand struct{}

// About returns information about a command
func (c ConfigCommand) About() alfred.CommandDef {
	return alfred.CommandDef{
		Keyword:     "config",
		Description: "Set workflow options",
		IsEnabled:   isAuthorized(),
	}
}

// Items returns a list of filter items
func (c ConfigCommand) Items(arg, dataa string) (items []alfred.Item, err error) {
	parts := alfred.TrimAllLeft(strings.SplitN(query, " ", 2))
	dlog.Printf("parts: %s", parts[0])

	if len(parts) == 1 {
		addItem := func(name, desc string) {
			if alfred.FuzzyMatches(name, arg) {
				items = append(items, alfred.NewKeywordItem(name, prefix, " ", desc))
			}
		}
		addItem("nest", "Select your default Nest")
		addItem("scale", "Select temperature scale used in this workflow")
	} else {
		property := parts[0]
		query = parts[1]

		switch property {
		case "nest":
			prefix += "nest "
			if err = checkRefresh(); err != nil {
				return
			}
			for _, t := range cache.AllData.Devices.Thermostats {
				if alfred.FuzzyMatches(t.Name, query) {
					data := configMessage{Property: "nest", Name: t.Name, DeviceId: t.DeviceId}
					dataString, _ := json.Marshal(data)
					items = append(items, alfred.MakeChoice(alfred.Item{
						Title:        t.Name,
						Autocomplete: prefix + t.Name,
						Arg:          "config " + string(dataString),
						SubtitleAll:  "ID: " + t.DeviceId,
					}, t.DeviceId == config.NestId))
				}
			}

		case "scale":
			prefix += property + " "
			items = append(items, getScaleItems(prefix, query, config.Scale)...)
		}
	}

	dlog.Printf("super sorting %v", items)
	items = alfred.SortItemsForKeyword(items, query)
	return
}

func (c ConfigCommand) Do(query string) (out string, err error) {
	var msg configMessage
	if err = json.Unmarshal([]byte(query), &msg); err != nil {
		return
	}

	switch msg.Property {
	case "nest":
		config.NestId = msg.DeviceId
		out = "Set default Nest to '" + msg.Name + "'"
	case "scale":
		config.Scale = msg.Scale
		if config.Scale == ScaleC {
			out = "Using Celsius scale"
		} else {
			out = "Using Fahrenheit scale"
		}
	}

	if out != "" {
		if err := alfred.SaveJson(configFile, &config); err != nil {
			dlog.Printf("Error saving cache: %s\n", err)
		}
	}

	return
}

type configMessage struct {
	Property string    `json:",omitempty"`
	Name     string    `json:",omitempty"`
	DeviceId string    `json:",omitempty"`
	Scale    TempScale `json:",omitempty"`
}
