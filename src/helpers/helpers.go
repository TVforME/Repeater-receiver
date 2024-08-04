package helpers

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConvertHzToMHz converts a frequency from Hz to MHz and returns a formatted string with three decimal places
func ConvertHzToMHz(frequency uint32) float64 {
	return float64(frequency) / 1000000.0
}

// DeleteAdapterFiles deletes the JSON and log files for a given adapter number
func DeleteAdapterFiles(adapterIndex int, dir string) {
	files := []string{
		fmt.Sprintf(filepath.Join(dir, "adapter%d.json"), adapterIndex),
		fmt.Sprintf(filepath.Join(dir, "adapter%d.txt"), adapterIndex),
		fmt.Sprintf(filepath.Join(dir, "service%d.json"), adapterIndex),
	}

	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				log.Printf("Error removing file %s: %v", file, err)
			}
		}
	}
}

// Function to read the output file and extract the modulation type
func GetModulationTypeFromFile(outputFile string) (string, error) {

	for {
		fileInfo, err := os.Stat(outputFile)
		if err == nil && fileInfo.Size() > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	file, err := os.Open(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to open output file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "--modulation") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "--modulation" && i+1 < len(parts) {
					return parts[i+1], nil
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading output file: %w", err)
	}

	return "", fmt.Errorf("modulation type not found in output")
}
