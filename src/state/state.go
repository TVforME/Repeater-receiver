package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/TVforME/Repeater-receiver/src/config"
	"github.com/TVforME/Repeater-receiver/src/frontend"
	"github.com/TVforME/Repeater-receiver/src/helpers"
	"github.com/TVforME/Repeater-receiver/src/service"
)

const (
	LISTENING = 0
	ANALYSING = 1
	FINDPIDS  = 2
	STREAMING = 3
	STOPPING  = 4
)

const maxRetries = 3

var LockStatusMu sync.Mutex
var AdapterStatusMap = make(map[int]CombinedStats)

// OsdStats represents the structure of the tuning parameters and signal stats
type OsdStats struct {
	DeliverySystem string  `json:"delivery-system"`
	Modulation     string  `json:"modulation"`
	SignalLock     bool    `json:"signal_lock"`
	CarrierLock    bool    `json:"carrier_lock"`
	ViterbiLock    bool    `json:"viterbi_lock"`
	SyncLock       bool    `json:"sync_lock"`
	Lock           bool    `json:"fe_lock"`
	State          uint8   `json:"state"`
	Signal         uint16  `json:"rssi"`
	SNR            uint16  `json:"snr"`
	BER            uint32  `json:"ber"`
	UNC            uint32  `json:"unc"`
	Frequency      float64 `json:"freq"`
	Bandwidth      uint8   `json:"bw"`
	SymbolRate     uint32  `json:"sr"`
}

// CombinedStats represents the combined structure of OsdStats and Service
type CombinedStats struct {
	OsdStats
	service.Service
}

// StateMachine represents the state machine for the set-top box
type StateMachine struct {
	AdapterIndex    int
	Config          config.DVBConfig
	Service         service.Service
	OsdStats        OsdStats
	tspCmd          *exec.Cmd
	Wg              *sync.WaitGroup
	Mu              sync.Mutex
	StopMonitorChan chan bool
	Debug           bool
}

// The main state machine
func (sm *StateMachine) Start() {
	defer sm.Wg.Done()

	for {
		if err := sm.AwaitSignal(); err != nil {
			sm.StopStreaming()
			log.Printf("Error in awaiting signal: %v", err)
			os.Exit(1)
		}

		if err := sm.FindServices(); err != nil {
			sm.StopStreaming()
			log.Printf("Error in getting services and pid info: %v", err)
			continue
		}

		if err := sm.StartStreaming(); err != nil {
			sm.StopStreaming()
			log.Printf("Error in streaming: %v", err)
			continue
		}

		break
	}

	sm.StopMonitorChan = make(chan bool)
	sm.MonitorSignal()
}

// This function sets up the front end determined on config settings
func fontendCmdArgs(sm *StateMachine) []string {
	frequency := sm.Config.Frequency

	var cmdArgs []string

	switch sm.Config.Type {
	case "DVB-T":
		frequency -= config.FreqOffset
		cmdArgs = []string{
			"--bandwidth", fmt.Sprintf("%d", sm.Config.Bandwidth),
		}
	case "DVB-T2":
		frequency -= config.FreqOffset
		cmdArgs = []string{
			"--bandwidth", fmt.Sprintf("%d", sm.Config.Bandwidth),
			"--delivery-system", "DVB-T2",
		}
	case "DVB-S":
		cmdArgs = []string{
			"--symbol-rate", fmt.Sprintf("%d", sm.Config.SymbolRate),
		}
	case "DVB-S2":
		cmdArgs = []string{
			"--symbol-rate", fmt.Sprintf("%d", sm.Config.SymbolRate),
			"--delivery-system", "DVB-S2",
		}
	case "DVB-S-Turbo":
		cmdArgs = []string{
			"--symbol-rate", fmt.Sprintf("%d", sm.Config.SymbolRate),
		}
	case "ATSC":
		cmdArgs = []string{
			"--delivery-system", "ATSC",
		}
	case "AUTO-T":
		frequency -= config.FreqOffset
		cmdArgs = []string{
			"--delivery-system", "undefined",
			"--bandwidth", fmt.Sprintf("%d", sm.Config.Bandwidth),
		}
	case "AUTO-S":
		cmdArgs = []string{
			"--delivery-system", "undefined",
			"--symbol-rate", fmt.Sprintf("%d", sm.Config.SymbolRate),
		}
	default:
		log.Fatalf("Unsupported DVB type: %s", sm.Config.Type)
	}

	cmdArgs = append([]string{"--frequency", fmt.Sprintf("%d", frequency)}, cmdArgs...)

	return cmdArgs
}

/*
Listens for Signal on selected frequency and system. Once, we receive a valid TS,
we create the adaptor(x).json file to use to determine services awailable.
*/
func (sm *StateMachine) AwaitSignal() error {

	// Set OSD state to Listening.
	sm.updateState(LISTENING)
	sm.updateAdapterStatus()

	if sm.Debug {
		log.Printf("==============================\n")
		log.Printf(" adapter %d ==> Awaiting Lock \n", sm.AdapterIndex)
		log.Printf("==============================\n")
	}

	// Tune frontend and begin analysing TS.
	cmdArgs := fontendCmdArgs(sm)

	analyseArgs := []string{"--signal-timeout", "0",
		"-P", "until", "-s", "2",
		"-P", "analyze", "--json", "-o", fmt.Sprintf(filepath.Join(config.TmpDir, "adapter%d.json"), sm.AdapterIndex),
		"-O", "drop",
	}

	// Combine all arguments into a single slice
	allArgs := append([]string{"-I", "dvb", "--adapter", fmt.Sprintf("%d", sm.AdapterIndex)}, append(cmdArgs, analyseArgs...)...)

	// Create the command
	cmd := exec.Command("tsp", allArgs...)

	// Build the debug string representation of the command
	debugCmdStr := "tsp " + strings.Join(allArgs, " ")

	cmd.Stderr = os.Stderr

	if sm.Debug {
		cmd.Stdout = os.Stdout
	} else {
		cmd.Stdout = nil
	}

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run tsp command %s: %w", debugCmdStr, err)
	}

	return nil
}

/*
Get First Service and begin finding the PCR,VIDEO, AUDIO and Video Type and Audio Type
for the service.
Awaits for TSP to produce adapter(x).json which is analsyed for the
services in the TS. If services > 1 then, selects the first service.
*/

func (sm *StateMachine) FindServices() error {

	// set sate runner to anaylsing
	sm.updateState(ANALYSING)
	sm.updateAdapterStatus()

	/*
		Find the first service in MPTS or SPTS.
		Fuction is able to select by index however.
	*/

	if err := sm.GetService(0); err != nil {
		log.Printf("error finding services in TS: %v", err)
	}

	cmdArgs := fontendCmdArgs(sm)

	// Additional arguments for streaming
	analyseArgs := []string{
		"-P", "zap", "-n", fmt.Sprintf("%d", sm.Service.ID), "-a", "eng",
		"-P", "until", "-s", "2",
		"-P", "analyze", "--json", "-o", fmt.Sprintf(filepath.Join(config.TmpDir, "service%d.json"), sm.AdapterIndex),
		"-O", "drop",
	}

	cmd := exec.Command("tsp",
		append([]string{"-v", "-I", "dvb", "--adapter",
			fmt.Sprintf("%d", sm.AdapterIndex)},
			append(cmdArgs, analyseArgs...)...)...)

	// Write  tsp output to adapter(x).txt file to use to determine system receive parameters.
	txtFile := filepath.Join(config.TmpDir, fmt.Sprintf("adapter%d.txt", sm.AdapterIndex))
	file, err := os.Create(txtFile)
	if err != nil {
		return fmt.Errorf("failed to create %s file: %w", txtFile, err)
	}
	defer file.Close()

	cmd.Stdout = file
	cmd.Stderr = file

	sm.tspCmd = cmd
	return cmd.Start()
}

// StartStreaming starts receiving the service and streams via RTP multicast
func (sm *StateMachine) StartStreaming() error {

	// Read the file and extract the modulation type
	modulation, merr := helpers.GetModulationTypeFromFile(filepath.Join(config.TmpDir, fmt.Sprintf("adapter%d.txt", sm.AdapterIndex)))
	if merr != nil {
		log.Fatalf("Error getting modulation type: %v", merr)
	}
	// show tuning stats in OSD
	sm.setInitialOsdStats(modulation)

	// set state runner to find pids
	sm.updateState(FINDPIDS)
	sm.updateAdapterStatus()

	// We should have a services(x).json file to check and anaylse for service PIDS
	err := sm.CheckForPIDFile()
	if err != nil {
		log.Fatal(err)
	}

	if sm.Debug {
		log.Printf("=============================================\n")
		log.Printf(" adapter %d ==> Streaming to %s:%d\n", sm.AdapterIndex, sm.Config.IP, sm.Config.Port)
		log.Printf(" Service-ID %d Service-Name %s \n", sm.Service.ID, sm.Service.Name)
		log.Printf(" PCR-PID %d \n", sm.Service.PCRPID)
		log.Printf(" Video-PID %d  (%s) \n", sm.Service.VideoPID, sm.Service.VideoType)
		log.Printf(" Audio-PID %d  (%s) \n", sm.Service.AudioPID, sm.Service.AudioType)
		log.Printf("=============================================\n")
	}
	cmdArgs := fontendCmdArgs(sm)

	// Additional arguments for streaming
	streamArgs := []string{
		"-P", "zap", "-n", fmt.Sprintf("%d", sm.Service.ID), "-a", "eng",
		"-O", "ip", "-e", "-f", "-r", fmt.Sprintf("%s:%d", sm.Config.IP, sm.Config.Port),
	}

	cmd := exec.Command("tsp",
		append([]string{"-v", "-I", "dvb", "--adapter",
			fmt.Sprintf("%d", sm.AdapterIndex)},
			append(cmdArgs, streamArgs...)...)...)

	if sm.Debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	sm.tspCmd = cmd
	return cmd.Start()
}

// StopStreaming stops the current tsp command
func (sm *StateMachine) StopStreaming() error {
	if sm.tspCmd != nil && sm.tspCmd.Process != nil {
		if sm.Debug {
			log.Printf("=========================\n")
			log.Printf(" adapter %d ==> Stopping \n", sm.AdapterIndex)
			log.Printf("=========================\n")
		}
		return sm.tspCmd.Process.Kill()
	}
	return nil
}

// updateOsdStats updates the tuning stats from a frontend
func (sm *StateMachine) updateOsdStats(status frontend.FrontendStatus, signalStrength, snr uint16, ber uint32) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sm.OsdStats.SignalLock = status&frontend.FE_HAS_SIGNAL != 0
	sm.OsdStats.CarrierLock = status&frontend.FE_HAS_CARRIER != 0
	sm.OsdStats.ViterbiLock = status&frontend.FE_HAS_VITERBI != 0
	sm.OsdStats.SyncLock = status&frontend.FE_HAS_SYNC != 0
	sm.OsdStats.Lock = status&frontend.FE_HAS_LOCK != 0

	if sm.OsdStats.Lock && sm.OsdStats.CarrierLock {
		sm.OsdStats.Signal = signalStrength
		sm.OsdStats.SNR = snr
		sm.OsdStats.BER = ber
	}
}

// MonitorSignal monitors the signal using the frontend library
func (sm *StateMachine) MonitorSignal() {

	// set state runner to streaming
	sm.updateState(STREAMING)
	sm.updateAdapterStatus()

	if sm.Debug {
		log.Printf("=================================\n")
		log.Printf(" adapter %d ==> Monitoring Stats \n", sm.AdapterIndex)
		log.Printf("=================================\n")
	}
	frontendFile, err := frontend.OpenFrontend(sm.AdapterIndex)
	if err != nil {
		log.Fatalf("Failed to open frontend device: %v\n", err)
		return
	}
	defer frontendFile.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	lockLost := make(chan bool)

	go sm.monitorSignalLoop(frontendFile, lockLost)

	for {
		select {
		case <-lockLost:
			sm.handleLockLost()
			return
		case <-sigChan:
			log.Println("Received termination signal, shutting down gracefully...")
			close(sm.StopMonitorChan)
			return
		default:
			sm.updateAdapterStatus()
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (sm *StateMachine) setInitialOsdStats(modulation string) {
	sm.OsdStats.Frequency = helpers.ConvertHzToMHz(sm.Config.Frequency)
	sm.OsdStats.DeliverySystem = sm.Config.Type
	sm.OsdStats.Bandwidth = sm.Config.Bandwidth
	sm.OsdStats.SymbolRate = sm.Config.SymbolRate
	sm.OsdStats.Modulation = modulation
}

// Function to get stats with retry logic
func getStatsWithRetries(frontendFile *os.File) (frontend.FrontendStatus, uint16, uint16, uint32, error) {
	var status frontend.FrontendStatus
	var signalStrength, snr uint16
	var ber uint32
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		status, signalStrength, snr, ber, err = frontend.GetStats(frontendFile)
		if err == nil {
			return status, signalStrength, snr, ber, nil
		}
		if attempt < maxRetries {
			time.Sleep(1 * time.Second)
		}
	}

	return status, signalStrength, snr, ber, err
}

func (sm *StateMachine) monitorSignalLoop(frontendFile *os.File, lockLost chan bool) {
	lockCount := 0
	time.Sleep(500 * time.Millisecond)

	for {
		status, signalStrength, snr, ber, err := getStatsWithRetries(frontendFile)
		if err != nil {
			log.Printf("Failed to get stats after %d attempts: %v\n", maxRetries, err)
			lockLost <- true
			return
		}

		sm.updateOsdStats(status, signalStrength, snr, ber)

		if sm.OsdStats.Lock && sm.OsdStats.CarrierLock {
			lockCount = 0
		} else {
			lockCount++
			if lockCount >= 10 {
				sm.updateState(STOPPING)
				sm.clearOsdStats()
				sm.updateAdapterStatus()
				time.Sleep(500 * time.Millisecond)
				lockLost <- true
				return
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (sm *StateMachine) handleLockLost() {
	sm.StopStreaming()
	helpers.DeleteAdapterFiles(sm.AdapterIndex, config.TmpDir)
	sm.Start()
}

func (sm *StateMachine) updateAdapterStatus() {
	sm.Mu.Lock()
	OsdStats := sm.OsdStats
	service := sm.Service
	sm.Mu.Unlock()

	combinedStats := CombinedStats{
		OsdStats: OsdStats,
		Service:  service,
	}

	LockStatusMu.Lock()
	AdapterStatusMap[sm.AdapterIndex] = combinedStats
	LockStatusMu.Unlock()
}

func (sm *StateMachine) updateState(state uint8) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	sm.OsdStats.State = state
}

func (sm *StateMachine) clearOsdStats() {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sm.OsdStats.SignalLock = false
	sm.OsdStats.SyncLock = false
	sm.OsdStats.CarrierLock = false
	sm.OsdStats.ViterbiLock = false
	sm.OsdStats.Lock = false
	sm.OsdStats.Modulation = ""
	sm.OsdStats.Signal = 0
	sm.OsdStats.SNR = 0
	sm.OsdStats.BER = 0
}

func (sm *StateMachine) CheckForPIDFile() error {
	filename := fmt.Sprintf(filepath.Join(config.TmpDir, "service%d.json"), sm.AdapterIndex)
	// Await till file has data before analysis
	for {
		fileInfo, err := os.Stat(filename)
		if err == nil && fileInfo.Size() > 0 {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	service, err := service.ExtractService(filename)
	if err != nil {
		return fmt.Errorf("error extracting service: %v", err)
	}

	sm.Service = service

	return nil
}

// returns service id and name from the JSON list by index.
// GetService returns the service id and name from the JSON list by index.
func (sm *StateMachine) GetService(serviceIndex uint8) error {
	filename := fmt.Sprintf(filepath.Join(config.TmpDir, "adapter%d.json"), sm.AdapterIndex)

	// Await till file has data before opening.
	for {
		fileInfo, err := os.Stat(filename)
		if err == nil && fileInfo.Size() > 0 {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	// Just wait to allow closing file
	time.Sleep(100 * time.Millisecond)

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return err
	}

	// Check if there are clear services
	pids, ok := result["pids"].([]interface{})
	if !ok || len(pids) == 0 {
		return fmt.Errorf("no [pids] section found in adapter%d.json", sm.AdapterIndex)
	}

	var serviceCount int
	for _, pid := range pids {
		pidMap, ok := pid.(map[string]interface{})
		if !ok {
			continue
		}
		if count, ok := pidMap["service-count"].(float64); ok {
			serviceCount += int(count)
		}
	}

	if serviceCount == 0 {
		return fmt.Errorf("no clear services found in adapter%d.json", sm.AdapterIndex)
	}

	// OK we have a service count > 0 therefore let's get the list of services
	servicesList, ok := result["services"].([]interface{})
	if !ok || len(servicesList) == 0 {
		return fmt.Errorf("no services found in adapter%d.json", sm.AdapterIndex)
	}

	if int(serviceIndex) >= len(servicesList) {
		return fmt.Errorf("service index %d out of range, total services: %d", serviceIndex, len(servicesList))
	}

	selectedService, ok := servicesList[serviceIndex].(map[string]interface{})
	if !ok {
		return fmt.Errorf("selected service is not a valid map structure")
	}

	if id, ok := selectedService["id"].(float64); ok {
		sm.Service.ID = int(id)
	} else {
		return fmt.Errorf("service id is not a valid number")
	}

	if name, ok := selectedService["name"].(string); ok {
		sm.Service.Name = name
	} else {
		return fmt.Errorf("service name is not a valid string")
	}

	if sm.Debug {
		fmt.Printf("Service ID: %d Name: %s\n", sm.Service.ID, sm.Service.Name)
	}

	return nil
}
