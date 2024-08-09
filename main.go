package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"s7_plc_read/utils" // Replace with the actual module path

	"github.com/robinson/gos7"
)

var (
	plcDataMutex sync.RWMutex
	plcData      utils.PLCData
)

func main() {
	// Load config
	utils.LoadConfig("config.json")

	// Check if PLC is reachable
	plcReachable := utils.IsReachable(utils.ConfigData.PlcIP, utils.ConfigData.PlcPort)
	if !plcReachable {
		log.Fatalf("PLC at %s:%s is not reachable", utils.ConfigData.PlcIP, utils.ConfigData.PlcPort)
	}

	// Wait for the PLC to become reachable
	utils.WaitForPLC(utils.ConfigData.PlcIP, utils.ConfigData.PlcPort, 5*time.Second)

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
				plcDataMutex.Lock()
				plcData = utils.MapBytesToPLCData(data)
				plcDataMutex.Unlock()

				// Print the data for debugging purposes
				fmt.Printf("PLC Data - Tag1: %d, Tag2: %d, Tag3: %d, Tag4: %d\n", plcData.Tag1, plcData.Tag2, plcData.Tag3, plcData.Tag4)

			case <-done:
				return
			}
		}
	}()

	// Start the HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		plcDataMutex.RLock()
		defer plcDataMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(plcData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	go func() {
		log.Println("Starting server on :9999")
		if err := http.ListenAndServe(":9999", nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down...")

	// Stop the goroutine
	done <- true
}
