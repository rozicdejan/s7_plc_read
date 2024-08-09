package utils

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type Config struct {
	PlcIP           string `json:"PlcIP"`
	InfluxDBURL     string `json:"InfluxDBURL"`
	InfluxDBHealth  string `json:"InfluxDBHealth"`
	InfluxDBToken   string `json:"InfluxDBToken"`
	InfluxDBOrg     string `json:"InfluxDBOrg"`
	InfluxDBBucket  string `json:"InfluxDBBucket"`
	ReconnectDelay  int    `json:"ReconnectDelay"` // In seconds
	PlcPort         string `json:"PlcPort"`
	WriteToInfluxDB bool   `json:"WriteToInfluxDB"` // New field for enabling/disabling InfluxDB writing
	WebServer       bool   `json:"WebServer"`       // New field for enabling/disabling the web server
}

var ConfigData Config

// Check if file exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func LoadConfig(filePath string) {

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	err = json.Unmarshal(data, &ConfigData)
	if err != nil {
		log.Fatalf("Failed to unmarshal config file: %v", err)
	}
}

func GetReconnectDelay() time.Duration {
	return time.Duration(ConfigData.ReconnectDelay) * time.Second
}
