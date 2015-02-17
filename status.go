package main

import (
	"fmt"

	"github.com/jason0x43/go-alfred"
)

type StatusCommand struct{}

func (t StatusCommand) Keyword() string {
	return "status"
}

func (t StatusCommand) IsEnabled() bool {
	return isAuthorized()
}

func (t StatusCommand) MenuItem() alfred.Item {
	if config.NestId == "" {
		return alfred.Item{
			Title:        "Select a default Nest",
			Autocomplete: "config nest" + alfred.Separator + " ",
			Valid:        alfred.Invalid,
		}
	} else {
		if err := checkRefresh(); err != nil {
			return alfred.Item{
				Title:       "Error communicating with Nest",
				SubtitleAll: fmt.Sprintf("%v", err),
				Valid:       alfred.Invalid,
			}
		} else {
			thermostat, _ := cache.AllData.Devices.Thermostats[config.NestId]
			structure, _ := cache.AllData.Structures[thermostat.StructureId]
			return alfred.Item{
				Title: thermostat.Name,
				SubtitleAll: fmt.Sprintf("Temp: %v, Humidity: %v, Mode: %v, Presence: %v",
					thermostat.AmbientTemperature(config.Scale), thermostat.Humidity, thermostat.HvacMode,
					structure.Away),
				Valid: alfred.Invalid,
			}
		}
	}
}

func (t StatusCommand) Items(prefix, query string) (items []alfred.Item, err error) {
	items = append(items, t.MenuItem())
	return
}
