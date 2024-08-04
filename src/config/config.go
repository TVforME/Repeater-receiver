package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"receiver/helpers"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// global debug flag and path to tmp dir and OSD freq offset
var (
	DebugFlag  bool
	TmpDir     string
	FreqOffset uint32
)

// NetworkConfig represents the network configuration
type NetworkConfig struct {
	StatsIP     string `yaml:"stats_ip" validate:"required,ip"`
	Port        int    `yaml:"port" validate:"required,min=1024,max=65535"`
	InterfaceIP string `yaml:"interface_ip" validate:"omitempty,ip"`
	WebIP       string `yaml:"web_ip" validate:"required,ip"`
}

type DvbtOffset struct {
	FreqOffset uint32 `yaml:"freq_offset" validate:"required,min=600000000,max=700000000"`
}

// DVBConfig represents the configuration for a single DVB type
type DVBConfig struct {
	Type       string `yaml:"type" validate:"required,oneof= 'AUTO-T' 'DVB-T' 'DVB-T2' 'DVB-S' 'DVB-S2' 'DVB-S-Turbo' 'AUTO-S' 'ATSC'"`
	Frequency  uint32 `yaml:"frequency" validate:"required,min=0"`
	Bandwidth  uint8  `yaml:"bandwidth,omitempty" validate:"min=0,max=8"`
	SymbolRate uint32 `yaml:"symbol_rate,omitempty" validate:"min=0"`
	IP         string `yaml:"ip" validate:"required,ip"`
	Port       int    `yaml:"port" validate:"required,min=1024,max=65535"`
}

// Config represents the configuration for the network and DVB-T and DVB-S
type Config struct {
	Network     NetworkConfig `yaml:"network" validate:"required"`
	Dvbt_offset DvbtOffset    `yaml:"dvbt_offset" validate:"required"`
	DVB         []DVBConfig   `yaml:"dvb" validate:"required,dive"`
}

// ReadConfig reads the configuration from a YAML file
func ReadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	FreqOffset = config.Dvbt_offset.FreqOffset

	return &config, nil
}

// ValidateConfig validates the configuration using go-playground/validator
func ValidateConfig(config *Config) error {
	validate := validator.New()
	return validate.Struct(config)
}

// KillRunningTSPProcesses checks for and kills any running tsp processes
func KillRunningTSPProcesses() error {
	cmd := exec.Command("pgrep", "tsp")
	output, err := cmd.Output()
	if err != nil {
		// No running tsp processes found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil
		}
		return err
	}

	pids := strings.Fields(string(output))
	for _, pid := range pids {
		killCmd := exec.Command("kill", "-9", pid)
		if err := killCmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

// CheckAdapters checks for the availability of DVB adapters and delete all adapter and service files
// If programmed happened to crash in a previous instance.
func CheckAdapters(maxAdapters int) []int {
	availableAdapters := []int{}
	for i := 0; i <= maxAdapters; i++ {
		if _, err := os.Stat(fmt.Sprintf("/dev/dvb/adapter%d", i)); err == nil {
			availableAdapters = append(availableAdapters, i)
			helpers.DeleteAdapterFiles(i, TmpDir)
		}
	}
	return availableAdapters
}

// CheckDependencies checks if tsduck is installed.
// We need TSduck cmd line to run.
func CheckDependencies() error {
	dependencies := []string{"tsp"}
	for _, dep := range dependencies {
		cmd := exec.Command(dep, "--version")
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("%s is not installed or not in PATH", dep)
		}
	}
	return nil
}

func SetupTempDir() {
	TmpDir = filepath.Join(os.TempDir(), "dvb")
	_, err := os.Stat(TmpDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(TmpDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create tmp directory: %v", err)
		}
	} else if err != nil {
		log.Fatalf("Failed to check tmp directory: %v", err)
	}
}
