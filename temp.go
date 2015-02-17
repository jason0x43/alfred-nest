package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"unicode"

	"github.com/jason0x43/go-alfred"
)

type TempCommand struct{}

func (t TempCommand) Keyword() string {
	return "temp"
}

func (t TempCommand) IsEnabled() bool {
	return isAuthorized() && config.NestId != ""
}

func (t TempCommand) MenuItem() alfred.Item {
	return alfred.Item{
		Title:        t.Keyword(),
		Autocomplete: t.Keyword() + " ",
		SubtitleAll:  "View and adjust your default Nest’s temperature",
		Valid:        alfred.Invalid,
	}
}

func (t TempCommand) Items(prefix, query string) (items []alfred.Item, err error) {
	if err = checkRefresh(); err != nil {
		return
	}

	thermostat, ok := cache.AllData.Devices.Thermostats[config.NestId]
	if !ok {
		return items, errors.New("Couldn’t access your default Nest")
	}

	temp := thermostat.AmbientTemperature(config.Scale)

	if query != "" {
		if newVal, err := strconv.ParseFloat(query, 64); err == nil {
			if newVal != temp.Value() {
				newTemp := NewTemp(newVal, config.Scale)

				var changeType string
				if thermostat.HvacMode == ModeRange {
					if newTemp.Value() > temp.Value() {
						changeType = "Heat"
					} else {
						changeType = "Cool"
					}
				} else {
					mode := string(thermostat.HvacMode)
					changeType = string(unicode.ToUpper(rune(mode[0]))) + mode[1:]
				}

				data := tempMessage{
					DeviceId:   thermostat.DeviceId,
					TargetTemp: newVal,
					Scale:      config.Scale,
				}
				dataString, _ := json.Marshal(data)

				items = append(items, alfred.Item{
					Title:       fmt.Sprintf("%s to %s", changeType, newTemp),
					SubtitleAll: fmt.Sprintf("Current temperature is %s", temp),
					Arg:         "temp " + string(dataString),
				})
				return items, nil
			}
		}
	}

	var subtitle string

	if thermostat.HvacMode == ModeRange {
		targetHigh := thermostat.TargetTemperatureHigh(config.Scale)
		targetLow := thermostat.TargetTemperatureLow(config.Scale)
		subtitle = "Target is "
		subtitle += fmt.Sprintf("%s to %s", targetLow, targetHigh)
	} else {
		if thermostat.HvacMode == ModeHeat {
			subtitle = "Heating to "
		} else {
			subtitle = "Cooling to "
		}
		subtitle += thermostat.TargetTemperature(config.Scale).String()
	}

	items = append(items, alfred.Item{
		Title:       subtitle,
		SubtitleAll: fmt.Sprintf("Current temperature is %s", temp),
		Valid:       alfred.Invalid,
	})

	return items, nil
}

func (t TempCommand) Do(query string) (out string, err error) {
	var msg tempMessage
	if err = json.Unmarshal([]byte(query), &msg); err != nil {
		return
	}

	thermostat, ok := cache.AllData.Devices.Thermostats[msg.DeviceId]
	if !ok {
		return out, errors.New("Unknown thermostat '" + msg.DeviceId + "'")
	}

	session := OpenSession(config.AccessToken)
	var newTemp Temperature

	if thermostat.HvacMode == ModeRange {
		if msg.TargetTemp < thermostat.AmbientTemperature(config.Scale).Value() {
			newTemp, err = session.SetTargetTemp(msg.DeviceId, msg.Temperature(), TypeLow)
		} else {
			newTemp, err = session.SetTargetTemp(msg.DeviceId, msg.Temperature(), TypeHigh)
		}
	} else {
		newTemp, err = session.SetTargetTemp(msg.DeviceId, msg.Temperature(), "")
	}

	if err != nil {
		return
	}

	scheduleRefresh()

	return fmt.Sprintf("Set temperature to %s", newTemp), err
}

type tempMessage struct {
	DeviceId   string
	TargetTemp float64
	Scale      TempScale
}

func (t *tempMessage) Temperature() Temperature {
	return NewTemp(t.TargetTemp, t.Scale)
}
