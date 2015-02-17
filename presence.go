package main

import (
	"encoding/json"
	"fmt"

	"github.com/jason0x43/go-alfred"
)

type PresenceCommand struct{}

func (t PresenceCommand) Keyword() string {
	return "presence"
}

func (t PresenceCommand) IsEnabled() bool {
	return isAuthorized() && config.NestId != ""
}

func (t PresenceCommand) MenuItem() alfred.Item {
	return alfred.Item{
		Title:        t.Keyword(),
		Autocomplete: t.Keyword() + " ",
		SubtitleAll:  "Tell Nest whether you’re home or away",
		Valid:        alfred.Invalid,
	}
}

func (t PresenceCommand) Items(prefix, query string) (items []alfred.Item, err error) {
	thermostat, _ := cache.AllData.Devices.Thermostats[config.NestId]
	structure, _ := cache.AllData.Structures[thermostat.StructureId]

	addItem := func(a Presence, desc string) {
		if alfred.FuzzyMatches(string(a), query) {
			data := awayMessage{StructureId: structure.StructureId, Away: a}
			dataString, _ := json.Marshal(data)

			items = append(items, alfred.MakeChoice(alfred.Item{
				Title:        string(a),
				SubtitleAll:  desc,
				Autocomplete: prefix + string(a),
				Arg:          "away " + string(dataString),
			}, structure.Away == a))
		}
	}

	addItem(Home, "You’re at home")
	addItem(Away, "You’re away")
	addItem(AutoAway, "Let Nest figure out if you’re away")

	return
}

func (t PresenceCommand) Do(query string) (out string, err error) {
	var msg awayMessage
	if err = json.Unmarshal([]byte(query), &msg); err != nil {
		return
	}

	session := OpenSession(config.AccessToken)
	err = session.SetPresence(msg.StructureId, msg.Away)
	if err != nil {
		return
	}

	scheduleRefresh()

	return fmt.Sprintf("Set presence to %s", msg.Away), err
}

type awayMessage struct {
	StructureId string
	Away        Presence
}
