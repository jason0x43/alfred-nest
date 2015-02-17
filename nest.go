package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jason0x43/go-log"
)

const (
	ApiHost = "https://developer-api.nest.com"
)

type AllData struct {
	Metadata   Metadata             `json:"metadata"`
	Devices    Devices              `json:"devices"`
	Structures map[string]Structure `json:"structures"`
}

type Metadata struct {
	AccessToken   string `json:"access_token"`
	ClientVersion int64  `json:"client_version"`
}

type Devices struct {
	Thermostats map[string]Thermostat `json:"thermostats"`
}

type Thermostat struct {
	DeviceId               string    `json:"device_id"`
	Locale                 string    `json:"locale"`
	SoftwareVersion        string    `json:"software_version"`
	StructureId            string    `json:"structure_id"`
	Name                   string    `json:"name"`
	NameLong               string    `json:"name_long"`
	LastConnection         time.Time `json:"last_connection"`
	IsOnline               bool      `json:"is_online"`
	CanCool                bool      `json:"can_cool"`
	CanHeat                bool      `json:"can_heat"`
	IsUsingEmergencyHeat   bool      `json:"is_using_emergency_heat"`
	HasFan                 bool      `json:"has_fan"`
	FanTimerActive         bool      `json:"fan_timer_active"`
	FanTimerTimeout        time.Time `json:"fan_timer_timeout"`
	HasLeaf                bool      `json:"has_leaf"`
	TemperatureScale       TempScale `json:"temperature_scale"`
	TargetTemperatureF     TempF     `json:"target_temperature_f"`
	TargetTemperatureC     TempC     `json:"target_temperature_c"`
	TargetTemperatureHighF TempF     `json:"target_temperature_high_f"`
	TargetTemperatureHighC TempC     `json:"target_temperature_high_c"`
	TargetTemperatureLowF  TempF     `json:"target_temperature_low_f"`
	TargetTemperatureLowC  TempC     `json:"target_temperature_low_c"`
	AwayTemperatureHighF   TempF     `json:"away_temperature_high_f"`
	AwayTemperatureHighC   TempC     `json:"away_temperature_high_c"`
	AwayTemperatureLowF    TempF     `json:"away_temperature_low_f"`
	AwayTemperatureLowC    TempC     `json:"away_temperature_low_c"`
	HvacMode               HvacMode  `json:"hvac_mode"`
	AmbientTemperatureF    TempF     `json:"ambient_temperature_f"`
	AmbientTemperatureC    TempC     `json:"ambient_temperature_c"`
	Humidity               Humidity  `json:"humidity"`
}

type Structure struct {
	StructureId         string    `json:"structure_id"`
	Thermostats         []string  `json:"thermostats"`
	Away                Presence  `json:"away"`
	Name                string    `json:"name"`
	PeakPeriodStartTime time.Time `json:"peak_period_start_time"`
	PeakPeriodEndTime   time.Time `json:"peak_period_end_time"`
	TimeZone            string    `json:"time_zone"`
	Eta                 struct {
		TripId                      string    `json:"trip_id"`
		EstimatedArrivalWindowBegin time.Time `json:"estimated_arrival_window_begin"`
		EstimatedArrivalWindowEnd   time.Time `json:"estimated_arrival_window_end"`
	} `json:"eta"`
}

type Session struct {
	token string
}

type Presence string
type TempF float64
type TempC float64
type Humidity float64
type TempScale string
type HighLow string
type HvacMode string

const (
	ScaleC    = TempScale("C")
	ScaleF    = TempScale("F")
	TypeHigh  = HighLow("high")
	TypeLow   = HighLow("low")
	ModeHeat  = HvacMode("heat")
	ModeCool  = HvacMode("cool")
	ModeRange = HvacMode("heat-cool")
	ModeOff   = HvacMode("off")
	Away      = Presence("away")
	Home      = Presence("home")
	AutoAway  = Presence("auto-away")
)

type Temperature interface {
	Value() float64
	Scale() TempScale
	String() string
}

func NewTemp(value float64, scale TempScale) Temperature {
	if scale == ScaleF {
		return TempF(value)
	} else {
		return TempC(value)
	}
}

func (t TempF) Value() float64 {
	return float64(t)
}

func (t TempF) Scale() TempScale {
	return ScaleF
}

func (t TempF) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 64) + "°F"
}

func (t TempC) Value() float64 {
	return float64(t)
}

func (t TempC) Scale() TempScale {
	return ScaleC
}

func (t TempC) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 64) + "°C"
}

func (h Humidity) String() string {
	return strconv.FormatFloat(float64(h), 'f', -1, 64) + "%"
}

func OpenSession(token string) Session {
	return Session{token: token}
}

func (session *Session) GetAllData() (allData AllData, err error) {
	data, err := session.get("/")
	if err != nil {
		return
	}

	dec := json.NewDecoder(strings.NewReader(data))
	err = dec.Decode(&allData)
	return
}

func (session *Session) GetThermostats() (thermostats []Thermostat, err error) {
	data, err := session.get("/thermostats")
	if err != nil {
		return thermostats, err
	}

	var s Devices
	dec := json.NewDecoder(strings.NewReader(data))
	if err := dec.Decode(&s); err != nil {
		return thermostats, err
	}

	for _, d := range s.Thermostats {
		thermostats = append(thermostats, d)
	}

	return
}

func (session *Session) IsAway(structureId string) (bool, error) {
	contents, err := session.get("/structures/" + structureId)
	if err != nil {
		return false, err
	}

	var s Structure
	dec := json.NewDecoder(strings.NewReader(contents))
	if err := dec.Decode(&s); err != nil {
		return false, err
	}

	return s.Away != "home", nil
}

func (session *Session) SetTargetTemp(nestId string, temp Temperature, hilo HighLow) (t Temperature, err error) {
	path := fmt.Sprintf("/devices/thermostats/%s/target_temperature_", nestId)
	if hilo != "" {
		path += string(hilo) + "_"
	}
	path += strings.ToLower(string(temp.Scale()))
	data, _ := json.Marshal(temp)

	var resp string
	if resp, err = session.put(path, data); err != nil {
		return
	}

	val, err := strconv.ParseFloat(resp, 64)
	if err != nil {
		return
	}

	return NewTemp(val, temp.Scale()), nil
}

func (session *Session) SetPresence(structureId string, presence Presence) (err error) {
	path := fmt.Sprintf("/structures//%s/away", structureId)
	data, _ := json.Marshal(presence)

	var resp string
	if resp, err = session.put(path, data); err != nil {
		return
	}

	log.Printf("got response: %s", resp)

	return nil
}

func (t *Thermostat) TemperatureScaleName() string {
	if t.TemperatureScale == ScaleF {
		return "Fahrenheit"
	} else {
		return "Celsius"
	}
}

func (t *Thermostat) SetTemperatureScale(name string) (err error) {
	lname := strings.ToLower(name)
	if lname == "fahrenheit" {
		t.TemperatureScale = ScaleF
	} else if lname == "celsius" {
		t.TemperatureScale = ScaleC
	} else {
		err = errors.New("Invalid temperature scale '" + name + "'")
	}
	return
}

func (t *Thermostat) TargetTemperature(scale TempScale) Temperature {
	switch scale {
	case ScaleF:
		return t.TargetTemperatureF
	case ScaleC:
		return t.TargetTemperatureC
	default:
		return t.TargetTemperature(t.TemperatureScale)
	}
}

func (t *Thermostat) TargetTemperatureHigh(scale TempScale) Temperature {
	switch scale {
	case ScaleF:
		return t.TargetTemperatureHighF
	case ScaleC:
		return t.TargetTemperatureHighC
	default:
		return t.TargetTemperatureHigh(t.TemperatureScale)
	}
}

func (t *Thermostat) TargetTemperatureLow(scale TempScale) Temperature {
	switch scale {
	case ScaleF:
		return t.TargetTemperatureLowF
	case ScaleC:
		return t.TargetTemperatureLowC
	default:
		return t.TargetTemperatureLow(t.TemperatureScale)
	}
}

func (t *Thermostat) AwayTemperatureHigh(scale TempScale) Temperature {
	switch scale {
	case ScaleF:
		return t.AwayTemperatureHighF
	case ScaleC:
		return t.AwayTemperatureHighC
	default:
		return t.AwayTemperatureHigh(t.TemperatureScale)
	}
}

func (t *Thermostat) AwayTemperatureLow(scale TempScale) Temperature {
	switch scale {
	case ScaleF:
		return t.AwayTemperatureLowF
	case ScaleC:
		return t.AwayTemperatureLowC
	default:
		return t.AwayTemperatureLow(t.TemperatureScale)
	}
}

func (t *Thermostat) AmbientTemperature(scale TempScale) Temperature {
	switch scale {
	case ScaleF:
		return t.AmbientTemperatureF
	case ScaleC:
		return t.AmbientTemperatureC
	default:
		return t.AmbientTemperature(t.TemperatureScale)
	}
}

// support /////////////////////////////////////////////////////////////

var client = &http.Client{}

func (session *Session) rawRequest(method, uri string, data []byte, follow int) (out string, err error) {
	var request *http.Request
	if data != nil {
		request, err = http.NewRequest(method, uri, bytes.NewReader(data))
		request.Header.Add("Content-Type", "application/json")
	} else {
		request, err = http.NewRequest(method, uri, nil)
	}
	if err != nil {
		return
	}
	request.Header.Add("Accept", "application/json")

	log.Printf("request: %#v", request)

	var resp *http.Response
	if resp, err = client.Do(request); err != nil {
		return
	}
	defer resp.Body.Close()

	log.Printf("response: %#v\n", resp)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf(resp.Status)
	}

	if resp.StatusCode == 307 && follow > 0 {
		uri := resp.Header.Get("Location")
		return session.rawRequest(method, uri, data, follow-1)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (session *Session) request(method, path string, data []byte) (out string, err error) {
	q := url.Values{}
	q.Set("auth", session.token)

	reqUri := ApiHost + path + "?" + q.Encode()

	return session.rawRequest(method, reqUri, data, 3)
}

func (session *Session) get(path string) (string, error) {
	return session.request("GET", path, nil)
}

func (session *Session) put(path string, data []byte) (string, error) {
	return session.request("PUT", path, data)
}

func (session *Session) patch(path string, data []byte) (string, error) {
	return session.request("PATCH", path, data)
}
