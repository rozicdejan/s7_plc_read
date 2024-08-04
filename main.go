package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/robinson/gos7"
)

type PLCData struct {
	Tag1 byte
	Tag2 byte
	Tag3 byte
	Tag4 int32
}

// Exported constants
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

func main() {
	// Check if InfluxDB is accessible
	influxDbHealth := IsInfluxDBAccessible(InfluxDBHealth)
	if !influxDbHealth {
		log.Fatalf("InfluxDB at %s is not accessible or not ready", InfluxDBHealth)
	}
	fmt.Println("InfluxDB is accessible and ready")

	// Check if PLC is reachable
	plcReachable := IsReachable(PlcIP, PlcPort)
	if !plcReachable {
		log.Fatalf("PLC at %s:%s is not reachable", PlcIP, PlcPort)
	}
	fmt.Println("PLC is reachable")

	// Define the PLC connection parameters
	handler := gos7.NewTCPClientHandler(PlcIP, 0, 1)
	handler.Timeout = 5 * time.Second
	handler.IdleTimeout = 5 * time.Second

	// Connect to the PLC
	err := handler.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to PLC: %v", err)
	}
	defer handler.Close()

	// Create a new PLC client
	client := gos7.NewClient(handler)

	// Create a new InfluxDB client
	influxClient := influxdb2.NewClient(InfluxDBURL, InfluxDBToken)
	defer influxClient.Close()
	writeAPI := influxClient.WriteAPIBlocking(InfluxDBOrg, InfluxDBBucket)

	// Create a ticker to read data every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Channel to handle system interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	done := make(chan bool, 1)

	go func() {
		for {
			select {
			case <-ticker.C:
				// Read data from DB1, starting at byte 0
				dbNumber := 1
				startAddress := 0
				numBytes := 7 // We need 7 bytes for our struct: 3 bytes for Tag1, Tag2, Tag3 and 4 bytes for Tag4
				data := make([]byte, numBytes)

				err := client.AGReadDB(dbNumber, startAddress, numBytes, data)
				if err != nil {
					log.Printf("Failed to read data from PLC: %v", err)
					continue
				}

				// Map the byte slice to the struct
				var plcData PLCData
				plcData.Tag1 = data[0]
				plcData.Tag2 = data[1]
				plcData.Tag3 = data[2]
				plcData.Tag4 = int32(binary.BigEndian.Uint32(data[3:7]))

				// Print the data
				fmt.Printf("PLC Data - Tag1: %d, Tag2: %d, Tag3: %d, Tag4: %d\n", plcData.Tag1, plcData.Tag2, plcData.Tag3, plcData.Tag4)

				// Create a new point and write it to InfluxDB
				p := influxdb2.NewPointWithMeasurement("temperature").
					AddTag("host", "plc").
					AddField("temperature1", plcData.Tag1).
					AddField("temperature2", plcData.Tag2).
					AddField("temperature3", plcData.Tag3).
					SetTime(time.Now())

				if err := writeAPI.WritePoint(context.Background(), p); err != nil {
					log.Printf("Failed to write data to InfluxDB: %v", err)
				}

			case <-done:
				return
			}
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down...")

	// Stop the goroutine
	done <- true
}
