package main

import (
	"log"
	"time"

	"github.com/jason0x43/go-alfred"
)

// isAuthorized returns true if this workflow has been authorized with
// Nest.com.
func isAuthorized() bool {
	if config.AccessToken == "" {
		return false
	}
	if time.Now().After(config.AccessExpiry) {
		return false
	}
	return true
}

// refresh downloads a user's current account data from Nest.com.
func refresh() error {
	log.Println("Getting status...")
	session := OpenSession(config.AccessToken)
	data, err := session.GetAllData()
	if err != nil {
		log.Println("Errror getting status:", err)
		return err
	}

	cache.AllData = data
	cache.Time = time.Now()
	alfred.SaveJson(cacheFile, &cache)
	configUpdated := false

	if config.NestId == "" {
		// if the user hasn't set a default Nest, pick the first one
		for id, _ := range cache.AllData.Devices.Thermostats {
			config.NestId = id
			configUpdated = true
			break
		}
	}

	if config.Scale == "" {
		// if the user hasn't set a scale, use the default Nest's
		if thermostat, ok := cache.AllData.Devices.Thermostats[config.NestId]; ok {
			config.Scale = TempScale(thermostat.TemperatureScale)
			configUpdated = true
		}
	}

	if configUpdated {
		if err := alfred.SaveJson(configFile, &config); err != nil {
			log.Printf("Error saving config: %s", err)
		}
	}

	return nil
}

// checkRefresh refreshes the cache if it hasn't been updated in the last 5
// minutes.
func checkRefresh() error {
	if time.Now().Sub(cache.Time).Minutes() < 5.0 {
		return nil
	}

	log.Println("Refreshing cache...")
	err := refresh()
	if err != nil {
		log.Println("Error refreshing cache:", err)
	}
	return err
}

// scheduleRefresh schedules a refresh on the next checkRefresh by setting the
// last update time to zero time.
func scheduleRefresh() error {
	cache.Time = time.Time{}
	return alfred.SaveJson(cacheFile, &cache)
}

// getThermostatByName searches the list of cached thermostats and returns the
// first one who's name matches a given name.
func getThermostatByName(name string) (Thermostat, bool) {
	for _, t := range cache.AllData.Devices.Thermostats {
		if t.Name == name {
			return t, true
		}
	}
	return Thermostat{}, false
}
