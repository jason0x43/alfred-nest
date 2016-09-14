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
)

const (
	nestAPI = "https://developer-api.nest.com"
)

// AllData is all the data returned for a user's account
type AllData struct {
	Metadata struct {
		AccessToken   string `json:"access_token"`
		ClientVersion int64  `json:"client_version"`
	} `json:"metadata"`
	Devices struct {
		Thermostats map[string]Thermostat
	} `json:"thermostats"`
	Structures map[string]Structure `json:"structures"`
}

// Thermostat is an individual thermostat
type Thermostat struct {
	DeviceID               string    `json:"device_id"`
	Locale                 string    `json:"locale"`
	SoftwareVersion        string    `json:"software_version"`
	StructureID            string    `json:"structure_id"`
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

// Structure is a structure that may contain multiple devices
type Structure struct {
	StructureID         string    `json:"structure_id"`
	Thermostats         []string  `json:"thermostats"`
	Away                Presence  `json:"away"`
	Name                string    `json:"name"`
	PeakPeriodStartTime time.Time `json:"peak_period_start_time"`
	PeakPeriodEndTime   time.Time `json:"peak_period_end_time"`
	TimeZone            string    `json:"time_zone"`
	Eta                 struct {
		TripID                      string    `json:"trip_id"`
		EstimatedArrivalWindowBegin time.Time `json:"estimated_arrival_window_begin"`
		EstimatedArrivalWindowEnd   time.Time `json:"estimated_arrival_window_end"`
	} `json:"eta"`
}

// Session is a session with the Nest API
type Session struct {
	token string
}

// Presence is home or away
type Presence string

// TempF is a temperature in Fahrenheit
type TempF float64

// TempC is a temperature in Celsius
type TempC float64

// Humidity is a relative humidity measurement
type Humidity float64

// TempScale indicates F or C
type TempScale string

// HighLow is a fan speed
type HighLow string

// HvacMode is a heat/cool mode
type HvacMode string

const (
	// ScaleC is the Celsius scale
	ScaleC = TempScale("C")
	// ScaleF is the Fahrenheit scale
	ScaleF = TempScale("F")
	// TypeHigh is high fan speed
	TypeHigh = HighLow("high")
	// TypeLow is low fan speed
	TypeLow = HighLow("low")
	// ModeHeat is heating mode
	ModeHeat = HvacMode("heat")
	// ModeCool is cooling mode
	ModeCool = HvacMode("cool")
	// ModeRange can heat or cool
	ModeRange = HvacMode("heat-cool")
	// ModeOff turns off the HVAC
	ModeOff = HvacMode("off")
	// Away means no one is home
	Away = Presence("away")
	// Home means someone is home
	Home = Presence("home")
	// AutoAway lets Nest detect if someone is home
	AutoAway = Presence("auto-away")
)

// Temperature is a temperature
type Temperature interface {
	Value() float64
	Scale() TempScale
	String() string
}

// NewTemp creates a temperature with a given value and scale
func NewTemp(value float64, scale TempScale) Temperature {
	if scale == ScaleF {
		return TempF(value)
	}
	return TempC(value)
}

// Value returns the float value of a TempF
func (t TempF) Value() float64 {
	return float64(t)
}

// Scale returns the unit scale of a TempF
func (t TempF) Scale() TempScale {
	return ScaleF
}

// String returns a formatted string value for a TempF
func (t TempF) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 64) + "°F"
}

// Value returns the float value of a TempC
func (t TempC) Value() float64 {
	return float64(t)
}

// Scale returns the unit scale of a TempC
func (t TempC) Scale() TempScale {
	return ScaleC
}

// String returns a formatted string value for a TempC
func (t TempC) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 64) + "°C"
}

// String returns a formatted string for a humidity value
func (h Humidity) String() string {
	return strconv.FormatFloat(float64(h), 'f', -1, 64) + "%"
}

// OpenSession opens a new Nest API session
func OpenSession(token string) Session {
	return Session{token: token}
}

// GetAllData rerieves all user data
func (session *Session) GetAllData() (allData AllData, err error) {
	data, err := session.get("/")
	if err != nil {
		return
	}

	err = json.NewDecoder(strings.NewReader(data)).Decode(&allData)
	return
}

// GetThermostats gets a list of all thermostats attached to a user account
func (session *Session) GetThermostats() (thermostats []Thermostat, err error) {
	data, err := session.get("/thermostats")
	if err != nil {
		return thermostats, err
	}

	var s Devices
	if err = json.NewDecoder(strings.NewReader(data)).Decode(&s); err != nil {
		return
	}

	for _, d := range s.Thermostats {
		thermostats = append(thermostats, d)
	}

	return
}

// IsAway indicates whether everyone is away from a structure
func (session *Session) IsAway(structureID string) (bool, error) {
	var contents []byte
	if contents, err = session.get("/structures/" + structureID); err != nil {
		return
	}

	var s Structure
	if err = json.NewDecoder(strings.NewReader(contents)).Decode(&s); err != nil {
		return
	}

	return s.Away != "home", nil
}

// SetTargetTemp sets the target temperature for a particular mode and Nest
func (session *Session) SetTargetTemp(nestID string, temp Temperature, hilo HighLow) (t Temperature, err error) {
	path := fmt.Sprintf("/devices/thermostats/%s/target_temperature_", nestID)
	if hilo != "" {
		path += string(hilo) + "_"
	}
	path += strings.ToLower(string(temp.Scale()))
	data, _ := json.Marshal(temp)

	var resp string
	if resp, err = session.put(path, data); err != nil {
		return
	}

	var val float64
	if val, err = strconv.ParseFloat(resp, 64); err != nil {
		return
	}

	return NewTemp(val, temp.Scale()), nil
}

// SetPresence sets the presence mode for a particular structure
func (session *Session) SetPresence(structureID string, presence Presence) (err error) {
	path := fmt.Sprintf("/structures//%s/away", structureID)
	data, _ := json.Marshal(presence)

	var resp string
	if resp, err = session.put(path, data); err != nil {
		return
	}

	dlog.Printf("got response: %s", resp)

	return nil
}

// TemperatureScaleName returns the name of the temperature scale being used by a specific Nest
func (t *Thermostat) TemperatureScaleName() string {
	if t.TemperatureScale == ScaleF {
		return "Fahrenheit"
	}
	return "Celsius"
}

// SetTemperatureScale sets the scale for a specific Nest
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

// TargetTemperature returns the target temperature in the given scale
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

// TargetTemperatureHigh returns the target high temperature in the given scale
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

// TargetTemperatureLow returns the target low temperature in the given scale
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

// AwayTemperatureHigh returns the target high away temperature in the given scale
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

// AwayTemperatureLow returns the target low away temperature in the given scale
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

// AmbientTemperature returns the current ambient temperature in the given scale
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

	dlog.Printf("request: %#v", request)

	var resp *http.Response
	if resp, err = client.Do(request); err != nil {
		return
	}
	defer resp.Body.Close()

	dlog.Printf("response: %#v\n", resp)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf(resp.Status)
	}

	if resp.StatusCode == 307 && follow > 0 {
		uri := resp.Header.Get("Location")
		return session.rawRequest(method, uri, data, follow-1)
	}

	var content []byte
	if content, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}

	return string(content), nil
}

func (session *Session) request(method, path string, data []byte) (out string, err error) {
	q := url.Values{}
	q.Set("auth", session.token)

	reqURI := nestAPI + path + "?" + q.Encode()

	return session.rawRequest(method, reqURI, data, 3)
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
