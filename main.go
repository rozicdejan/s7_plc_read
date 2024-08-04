package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"s7_plc_read/utils" // Replace with the actual module path

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/robinson/gos7"
)

type PLCData struct {
	Tag1 byte  `json:"Tag1"`
	Tag2 byte  `json:"Tag2"`
	Tag3 byte  `json:"Tag3"`
	Tag4 int32 `json:"Tag4"`
}

func main() {

	utils.LoadConfig("config.json")

	// Check if InfluxDB is accessible
	influxDbHealth := utils.IsInfluxDBAccessible(utils.ConfigData.InfluxDBHealth)
	if !influxDbHealth {
		log.Fatalf("InfluxDB at %s is not accessible or not ready", utils.ConfigData.InfluxDBHealth)
	}
	fmt.Println("InfluxDB is accessible and ready")
	// Check if PLC is reachable
	plcReachable := utils.IsReachable(utils.ConfigData.PlcIP, utils.ConfigData.PlcPort)
	if !plcReachable {
		log.Fatalf("PLC at %s:%s is not reachable", utils.ConfigData.PlcIP, utils.ConfigData.PlcPort)
	}
	fmt.Println("PLC is reachable")

	// Define the PLC connection parameters
	handler := gos7.NewTCPClientHandler(utils.ConfigData.PlcIP, 0, 1)
	handler.IdleTimeout = utils.GetReconnectDelay()

	// Connect to the PLC
	err := handler.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to PLC: %v", err)
	}
	defer handler.Close()

	// Create a new PLC client
	client := gos7.NewClient(handler)

	// Create a new InfluxDB client
	influxClient := influxdb2.NewClient(utils.ConfigData.InfluxDBURL, utils.ConfigData.InfluxDBToken)
	defer influxClient.Close()
	writeAPI := influxClient.WriteAPIBlocking(utils.ConfigData.InfluxDBOrg, utils.ConfigData.InfluxDBBucket)

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
				plcData := utils.MapBytesToPLCData(data)

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

//compile with go build -o my-go-app.exr main.go
