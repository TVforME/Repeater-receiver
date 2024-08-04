package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Service represents the structure of the JSON payload for services
type Service struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	PCRPID    int    `json:"pcr_pid"`
	VideoPID  int    `json:"video_pid"`
	AudioPID  int    `json:"audio_pid"`
	VideoType string `json:"video_type"`
	AudioType string `json:"audio_type"`
}

// PID represents the PID structure in the JSON
type PID struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Video       bool   `json:"video"`
	Audio       bool   `json:"audio"`
}

// TSDump represents the structure of the entire JSON payload
type TSDump struct {
	PIDs     []PID     `json:"pids"`
	Services []Service `json:"services"`
}

// extractService waits until the file exists and has data, then extracts service information
func ExtractService(filename string) (Service, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Service{}, err
	}
	defer file.Close()

	var tsDump TSDump

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&tsDump)
	if err != nil {
		return Service{}, err
	}

	// Assuming we want the first service in the list
	if len(tsDump.Services) == 0 {
		return Service{}, fmt.Errorf("no services found in the file")
	}

	service := tsDump.Services[0]

	// Update the service struct with PIDs
	for _, pid := range tsDump.PIDs {
		if pid.Video {
			service.VideoPID = pid.ID
			service.VideoType = determineVideoType(pid.Description)
		}
		if pid.Audio {
			service.AudioPID = pid.ID
			service.AudioType = determineAudioType(pid.Description)
		}
		if pid.Description == "PCR (not otherwise referenced)" {
			service.PCRPID = pid.ID
		}
	}

	// If PCRPID is 0, set it to VideoPID
	if service.PCRPID == 0 {
		service.PCRPID = service.VideoPID
	}

	return service, nil
}

// determineVideoType function to determine the video type based on the description
func determineVideoType(description string) string {
	switch {
	case strings.Contains(description, "1920x1080"):
		return "UHD"
	case strings.Contains(description, "1280x720"):
		return "HD"
	case strings.Contains(description, "720x576"):
		return "SD"
	default:
		return "?"
	}
}

// determineAudioType function to determine the audio type based on the description
func determineAudioType(description string) string {
	switch {
	case strings.Contains(description, "MPEG"):
		return "MPEG"
	case strings.Contains(description, "AAC"):
		return "AAC"
	case strings.Contains(description, "AC-3"):
		return "AC3"
	default:
		return "?"
	}
}
