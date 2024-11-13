package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Cache structure to store metrics and error statuses
type Cache struct {
	Metrics         []string // Raw Prometheus-formatted metrics
	ExitCode        int
	FileAccessError int
	ParseError      int
}

var (
	// Command-line flags
	interval = flag.Int("interval", 300, "Interval for metrics collection in seconds")
	timeout  = flag.Int("timeout", 200, "Timeout for scripts (in seconds)")
	labels   = flag.String("labels", "", "Additional labels for metrics")
	path     = flag.String("path", "/scripts", "Path to directory with bash scripts")
	prefix   = flag.String("prefix", "", "Prefix for metrics names")
	port     = flag.String("port", ":9000", "Port on which to expose metrics")

	// Cache and synchronization
	cache      = make(map[string]Cache)
	cacheMutex sync.RWMutex
)

// Function to execute a script and update its metrics in the cache
func updateScriptMetrics(scriptPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	output, err := cmd.Output()
	exitCode := 0
	parseError := 0

	if err != nil {
		// Get exit code if there was an error
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			log.Printf("Error executing script %s: %v", scriptPath, err)
			return
		}
	}

	var metrics []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		// Check if it's a valid Prometheus line (no empty lines, etc.)
		if line != "" {
			metrics = append(metrics, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error scanning output from %s: %v", scriptPath, err)
		parseError = 1
	}

	cacheMutex.Lock()
	cache[filepath.Base(scriptPath)] = Cache{Metrics: metrics, ExitCode: exitCode, ParseError: parseError}
	cacheMutex.Unlock()
}

// Function to find and execute all scripts in the specified directory
func updateAllMetrics() {
	scriptCount := 0
	err := filepath.Walk(*path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("File access error %s: %v", path, err)
			cacheMutex.Lock()
			cache[filepath.Base(path)] = Cache{FileAccessError: 1}
			cacheMutex.Unlock()
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".sh") {
			return nil
		}
		scriptCount++
		go updateScriptMetrics(path)
		return nil
	})

	// Exit the program with an error if no scripts are found
	if err == nil && scriptCount == 0 {
		log.Fatalf("No scripts found in directory %s", *path)
	}
}

// Function to convert metrics to Prometheus format
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8") // Set content type for Prometheus
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	for scriptName, cacheData := range cache {
		// Output script exit code, file access error, and JSON parsing error as separate metrics
		fmt.Fprintf(w, "script_exporter_error{error_name=\"script_exit_code\",script_name=\"%s\"} %d\n", scriptName, cacheData.ExitCode)
		fmt.Fprintf(w, "script_exporter_error{error_name=\"file_access_error\",script_name=\"%s\"} %d\n", scriptName, cacheData.FileAccessError)
		fmt.Fprintf(w, "script_exporter_error{error_name=\"json_parse_error\",script_name=\"%s\"} %d\n", scriptName, cacheData.ParseError)

		// Output the metrics directly as they are in Prometheus format
		for _, metric := range cacheData.Metrics {
			fmt.Fprintf(w, "%s\n", metric)
		}
	}
}

func main() {
	// Parse command-line flags
	flag.Parse()

	// Start cache update loop in background
	go func() {
		for {
			updateAllMetrics()
			time.Sleep(time.Duration(*interval) * time.Second)
		}
	}()

	// Configure and start HTTP server
	http.HandleFunc("/metrics", metricsHandler)
	log.Printf("Starting server on %s", *port)
	if err := http.ListenAndServe(*port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
