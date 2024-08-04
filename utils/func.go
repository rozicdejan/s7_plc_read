package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type PLCData struct {
	Tag1 byte
	Tag2 byte
	Tag3 byte
	Tag4 int32
}

/*
const (
	PlcIP          = "192.168.33.100"
	InfluxDBURL    = "http://192.168.107.100:8086"
	InfluxDBHealth = "http://192.168.107.100:8086/health"
	InfluxDBToken  = "xMnoCSdR_Z3JnzyXoYATsN_v0bsW0kUdFyix0cOcjgrhkKm-dFDCpzGmp8m20383fLRBlFWUfaDPyQbDpXPu4g=="
	InfluxDBOrg    = "DAFRA"
	InfluxDBBucket = "PLC_READ"
	ReconnectDelay = 5 * time.Second
	PlcPort        = "102"
)
*/

// IsReachable checks if a network address is reachable.
func IsReachable(ip string, port string) bool {
	_, err := net.DialTimeout("tcp", ip+":"+port, 3*time.Second)
	return err == nil
}

// IsInfluxDBAccessible checks if InfluxDB is accessible via HTTP.
func IsInfluxDBAccessible(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	message, ok := result["message"].(string)
	return ok && message == "ready for queries and writes"
}

// MapBytesToPLCData maps a byte slice to PLCData struct.
func MapBytesToPLCData(data []byte) PLCData {
	return PLCData{
		Tag1: data[0],
		Tag2: data[1],
		Tag3: data[2],
		Tag4: int32(binary.BigEndian.Uint32(data[3:7])),
	}
}

// waitForPLC waits until the PLC becomes reachable.
func WaitForPLC(ip, port string, delay time.Duration) {
	for {
		if IsReachable(ip, port) {
			fmt.Println("PLC is reachable")
			break
		}
		fmt.Println("Waiting for PLC to become reachable...")
		time.Sleep(delay)
	}
}

// waitForInfluxDB waits until InfluxDB becomes accessible.
func WaitForInfluxDB(url string, delay time.Duration) {
	for {
		if IsInfluxDBAccessible(url) {
			fmt.Println("InfluxDB is accessible and ready")
			break
		}
		fmt.Println("Waiting for InfluxDB to become accessible...")
		time.Sleep(delay)
	}
}
