package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/TVforME/Repeater-receiver/src/config"
	"github.com/TVforME/Repeater-receiver/src/net"
	"github.com/TVforME/Repeater-receiver/src/osd"
	"github.com/TVforME/Repeater-receiver/src/state"
)

const VERSION = "v1.0.1 Robert Hensel VK3DG"

// Main routine we start here.
func main() {

	// Command-line flags
	versionFlag := flag.Bool("v", false, "Print the version number and exit")
	versionFlagLong := flag.Bool("version", false, "Print the version number and exit (long form)")
	debugFlag := flag.Bool("d", false, "Enable debug mode")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// set global debug flag with local debug flag
	config.DebugFlag = *debugFlag

	if *versionFlag || *versionFlagLong {
		fmt.Printf("%s\n", VERSION)
		os.Exit(0)
	}

	if len(flag.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "Unknown argument: %s\n", flag.Args()[0])
		flag.Usage()
		os.Exit(1)
	}

	// Check if required dependencies are installed
	if err := config.CheckDependencies(); err != nil {
		log.Fatalf("Dependency check failed: %v", err)
	}

	// Kill off any running or zombie instances of tsp from previous error
	if err := config.KillRunningTSPProcesses(); err != nil {
		log.Fatalf("Failed to kill running tsp processes: %v", err)
	}

	settings, err := config.ReadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	if err := config.ValidateConfig(settings); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	availableAdapters := config.CheckAdapters(8)
	if len(availableAdapters) == 0 {
		log.Fatal("No DVB adapters available")
	}

	if len(availableAdapters) < len(settings.DVB) {
		log.Fatalf("Insufficient DVB adapters available. Needed: %d, Available: %d", len(settings.DVB), len(availableAdapters))
	}

	// Create dvb directory in tmp to keep adapter(x).json and service(x).json working files
	// Set up the temporary directory
	config.SetupTempDir()

	var stateMachines []*state.StateMachine
	var wg sync.WaitGroup

	// Spin up an instance of each type in config.
	for i, dvbConfig := range settings.DVB {
		wg.Add(1)
		sm := &state.StateMachine{
			AdapterIndex: availableAdapters[i],
			Config:       dvbConfig,
			Wg:           &wg,
			Debug:        config.DebugFlag,
		}

		stateMachines = append(stateMachines, sm)

		// Start up a go routine for each adapter
		go sm.Start()
	}

	// Spin up HTTP server for OSD
	wg.Add(1)
	go func() {
		defer wg.Done()
		osd.RunHTTPServer(settings.DVB, settings.Network)
	}()

	wg.Add(1)
	// spin up external udp to core link to determine when to switch to source.
	go func() {
		defer wg.Done()
		net.SendLockStats(settings.Network, stateMachines)
	}()

	// Handle termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received termination signal, shutting down gracefully...")
		for _, sm := range stateMachines {
			close(sm.StopMonitorChan)
			sm.StopStreaming()
		}
		os.Exit(0)
	}()

	wg.Wait()
}
