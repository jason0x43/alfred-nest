package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jason0x43/go-alfred"
)

type DevicesCommand struct{}

func (t DevicesCommand) Keyword() string {
	return "devices"
}

func (t DevicesCommand) IsEnabled() bool {
	return isAuthorized()
}

func (t DevicesCommand) MenuItem() alfred.Item {
	return alfred.Item{
		Title:        t.Keyword(),
		Autocomplete: t.Keyword() + " ",
		SubtitleAll:  "Control a specific Nest",
		Valid:        alfred.Invalid,
	}
}

func (t DevicesCommand) Items(prefix, query string) (items []alfred.Item, err error) {
	parts := strings.SplitN(query, alfred.Separator, 2)

	if err = checkRefresh(); err != nil {
		return
	}

	if len(parts) == 1 {
		for _, t := range cache.AllData.Devices.Thermostats {
			if alfred.FuzzyMatches(t.Name, query) {
				items = append(items, alfred.Item{
					Title:        t.Name,
					Autocomplete: prefix + t.Name + alfred.Separator + " ",
					SubtitleAll:  "ID: " + t.DeviceId,
					Valid:        alfred.Invalid,
				})
			}
		}
	} else {
		name := strings.TrimSpace(parts[0])
		thermostat, ok := getThermostatByName(name)
		if !ok {
			return items, errors.New("Unknown thermostat '" + name + "'")
		}

		prefix += name + alfred.Separator + " "
		return getDeviceItems(prefix, parts[1], thermostat.DeviceId)
	}

	return
}

func getDeviceItems(prefix, query, deviceId string) (items []alfred.Item, err error) {
	thermostat, ok := cache.AllData.Devices.Thermostats[deviceId]
	if !ok {
		return items, errors.New("Unknown device ID")
	}

	query = strings.TrimLeft(query, " ")
	parts := strings.SplitN(query, " ", 2)
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}

	addAdjustableItem := func(property string, value interface{}) {
		items = append(items, alfred.Item{
			Title:        property,
			Subtitle:     fmt.Sprintf("%v", value),
			Autocomplete: prefix + property + " ",
			Valid:        alfred.Invalid,
		})
	}

	addSelectableItem := func(property string, value interface{}) {
		items = append(items, alfred.Item{
			Title:        property,
			Subtitle:     fmt.Sprintf("%v", value),
			Autocomplete: prefix + property + " ",
			Valid:        alfred.Invalid,
		})
	}

	if len(parts) == 1 {
		if query == "" {
			// TODO: only show icon for top item; use leaf icon if leaf is active
			items = append(items, alfred.Item{
				Title:       thermostat.Name,
				SubtitleAll: fmt.Sprintf("ID: %v, SW: %v", thermostat.DeviceId, thermostat.SoftwareVersion),
				Valid:       alfred.Invalid,
			})

			var online string
			if thermostat.IsOnline {
				online = "Online"
			} else {
				online = "Offline"
			}

			items = append(items, alfred.Item{
				Title:       online,
				SubtitleAll: fmt.Sprintf("Last connected at %v", thermostat.LastConnection.Local().Format(time.RFC822)),
				Valid:       alfred.Invalid,
			})
		}

		if alfred.FuzzyMatches("scale", query) {
			addSelectableItem("scale", thermostat.TemperatureScaleName())
		}

		if alfred.FuzzyMatches("mode", query) {
			addSelectableItem("mode", thermostat.HvacMode)
		}

		if alfred.FuzzyMatches("away-low", query) {
			addAdjustableItem("away-low", thermostat.AwayTemperatureLow(config.Scale))
		}

		if alfred.FuzzyMatches("away-high", query) {
			addAdjustableItem("away-high", thermostat.AwayTemperatureHigh(config.Scale))
		}
	} else {
		addTempItem := func(name, newValue string, value interface{}) {
			if newValue != "" {
				if v, err := strconv.ParseFloat(parts[1], 64); err == nil {
					newTemp := NewTemp(v, config.Scale)
					items = append(items, alfred.Item{
						Title:       fmt.Sprintf("Set %s to %v", name, newTemp),
						SubtitleAll: fmt.Sprintf("Currently %v", value),
					})
				} else {
					items = append(items, alfred.Item{
						Title:       "Invalid temperature '" + newValue + "'",
						SubtitleAll: fmt.Sprintf("Currently %v", value),
					})
				}
			} else {
				addAdjustableItem(name, value)
			}
		}

		property := parts[0]
		switch strings.ToLower(property) {
		case "scale":
			prefix += property + " "
			items = append(items, getScaleItems(prefix, parts[1], thermostat.TemperatureScale)...)
		case "mode":
			prefix += property + " "
			items = append(items, getModeItems(prefix, parts[1], thermostat.HvacMode)...)
		case "away-low":
			prefix += property + " "
			addTempItem(property, parts[1], thermostat.AwayTemperatureLow(config.Scale))
		case "away-high":
			prefix += property + " "
			addTempItem(property, parts[1], thermostat.AwayTemperatureLow(config.Scale))
		}
	}

	return
}

type choiceMessage struct {
	Property string
	Value    interface{}
}

func getModeItems(prefix, query string, selected HvacMode) (items []alfred.Item) {
	addItem := func(mode HvacMode, desc string) {
		if alfred.FuzzyMatches(string(mode), query) {
			items = append(items, alfred.MakeChoice(alfred.Item{
				Title:        string(mode),
				SubtitleAll:  desc,
				Autocomplete: prefix + string(mode),
			}, selected == mode))
		}
	}
	addItem(ModeHeat, "Use the heater to maintain a minimum temperature")
	addItem(ModeCool, "Use the AC to maintain a maximum temperature")
	addItem(ModeRange, "Use both the heater and AC to maintain a temperature range")
	return alfred.SortItemsForKeyword(items, query)
}

func getScaleItems(prefix, query string, selected TempScale) (items []alfred.Item) {
	addItem := func(scale TempScale, desc string) {
		if alfred.FuzzyMatches(string(scale), query) {
			items = append(items, alfred.MakeChoice(alfred.Item{
				Title:        string(scale),
				SubtitleAll:  desc,
				Autocomplete: prefix + string(scale),
			}, selected == scale))
		}
	}
	addItem(ScaleC, "Use Celsius scale")
	addItem(ScaleF, "Use Fahrenheit scale")
	return alfred.SortItemsForKeyword(items, query)
}
