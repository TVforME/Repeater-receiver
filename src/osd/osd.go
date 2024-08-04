package osd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/TVforME/Repeater-receiver/src/config"
	"github.com/TVforME/Repeater-receiver/src/state"
	"github.com/TVforME/Repeater-receiver/src/systemstats"
)

// Reads a file from the static directory and returns its content as a string
func readFile(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Create the root html page.
func generateRootPageHTML(config []config.DVBConfig) (string, error) {
	htmlContent, err := readFile("static/root.html")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("root").Parse(htmlContent)
	if err != nil {
		return "", err
	}

	data := struct {
		AdapterLinks template.HTML
	}{
		AdapterLinks: template.HTML(generateAdapterLinks(config)),
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

// create adapter links
func generateAdapterLinks(dvb []config.DVBConfig) string {
	var links string
	for i := range dvb {
		links += fmt.Sprintf(`<p><a href="/adapter/%[1]d">Adapter %[1]d</a></p>`, i)
	}
	links += `<p><a href="/monitor">System Monitor</a></p>`
	return links
}

func generateAdapterHTML(adapterID int) (string, error) {
	htmlContent, err := readFile("static/adapter.html")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("adapter").Parse(htmlContent)
	if err != nil {
		return "", err
	}

	data := struct {
		AdapterID int
	}{
		AdapterID: adapterID,
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func generateMonitorPageHTML() (string, error) {
	htmlContent, err := readFile("static/monitor.html")
	if err != nil {
		return "", err
	}
	return htmlContent, nil
}

func RunHTTPServer(dvb []config.DVBConfig, net config.NetworkConfig) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		htmlContent, err := generateRootPageHTML(dvb)
		if err != nil {
			http.Error(w, "Error loading page", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlContent)
	})

	http.HandleFunc("/adapter/", func(w http.ResponseWriter, r *http.Request) {
		adapterIDStr := r.URL.Path[len("/adapter/"):]
		adapterID, err := strconv.Atoi(adapterIDStr)
		if err != nil {
			http.Error(w, "Invalid adapter ID", http.StatusBadRequest)
			return
		}

		htmlContent, err := generateAdapterHTML(adapterID)
		if err != nil {
			http.Error(w, "Error loading page", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlContent)
	})

	http.HandleFunc("/monitor", func(w http.ResponseWriter, r *http.Request) {
		htmlContent, err := generateMonitorPageHTML()
		if err != nil {
			http.Error(w, "Error loading page", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlContent)
	})

	http.HandleFunc("/monitor-sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for {
			stats, err := systemstats.GetSystemStats()
			if err != nil {
				http.Error(w, "Error retrieving system stats", http.StatusInternalServerError)
				return
			}

			data, err := json.Marshal(stats)
			if err != nil {
				http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
				return
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			time.Sleep(1 * time.Second)
		}
	})

	http.HandleFunc("/adapter-sse/", func(w http.ResponseWriter, r *http.Request) {
		adapterIDStr := r.URL.Path[len("/adapter-sse/"):]
		adapterID, err := strconv.Atoi(adapterIDStr)
		if err != nil {
			http.Error(w, "Invalid adapter ID", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for {
			state.LockStatusMu.Lock()
			adapterInfo, ok := state.AdapterStatusMap[adapterID]
			state.LockStatusMu.Unlock()

			if ok {
				data, err := json.Marshal(adapterInfo)
				if err != nil {
					http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}

			time.Sleep(1 * time.Second)
		}

	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	address := fmt.Sprintf("%s:%d", net.WebIP, 8080)
	if config.DebugFlag {
		log.Printf("Stats web Server started and listening on %s\n", address)
	}
	http.ListenAndServe(address, nil)
}
