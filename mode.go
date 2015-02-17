package main

import "github.com/jason0x43/go-alfred"

type ModeCommand struct{}

func (t ModeCommand) Keyword() string {
	return "mode"
}

func (t ModeCommand) IsEnabled() bool {
	return isAuthorized() && config.NestId != ""
}

func (t ModeCommand) MenuItem() alfred.Item {
	return alfred.Item{
		Title:        t.Keyword(),
		Autocomplete: t.Keyword() + " ",
		SubtitleAll:  "Set your Nest’s heat/cool mode",
		Valid:        alfred.Invalid,
	}
}

func (t ModeCommand) Items(prefix, query string) (items []alfred.Item, err error) {
	thermostat, _ := cache.AllData.Devices.Thermostats[config.NestId]
	return getModeItems(prefix, query, thermostat.HvacMode), nil
}
