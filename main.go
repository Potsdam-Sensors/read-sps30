package main

import (
	"fmt"
	"os"
	"time"

	"github.com/coreos/go-systemd/daemon"
)

func printlnf(format string, a ...interface{}) {
	println(fmt.Sprintf(format, a...))
}

const (
	ENV_VAR_SOCKET_PATH = "DATA_SOCKET"
	SAMPLE_PERIOD       = time.Second * 5
)

var dataSocketPath string

func main() {
	if v := os.Getenv(ENV_VAR_SOCKET_PATH); v == "" {
		printlnf("Environment variable needed, %s, was not set.", ENV_VAR_SOCKET_PATH)
		os.Exit(1)
	} else {
		dataSocketPath = v
	}
	printlnf("Data socket path set as \"%s\".", dataSocketPath)

	if len(os.Args) < 2 {
		printlnf("Expected at least one additional argument for port path.")
		os.Exit(1)
	}
	portPath := os.Args[1]
	port, err := openPort(portPath)
	if err != nil {
		printlnf("Failed to open port at given path, \"%s\", due to error: %v", portPath, err)
		os.Exit(1)
	}
	defer port.Close()

	if err := startSps30(port); err != nil {
		printlnf("Error starting sps30: %v", err)
		os.Exit(1)
	} else {
		println("Successfully started SPS30.")
	}

	sampleTicker := time.NewTicker(SAMPLE_PERIOD)
	defer sampleTicker.Stop()
	for {
		select {
		case <-sampleTicker.C:
			data, err := readSps30(port, time.Second*10)
			if err != nil {
				printlnf("Error reading sps30: %v", err)
				continue
			} else if data == nil {
				time.Sleep(time.Second)
				continue
			}
			printlnf(data.String())

			if err := data.SendGob(dataSocketPath); err != nil {
				printlnf("Failed to send data: %v", err)
			} else {
				println("Successfully sent data.")
			}
			daemon.SdNotify(false, "WATCHDOG=1")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
