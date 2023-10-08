package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mcuadros/go-syslog"
	"github.com/trivago/grok"
)

func loadPatterns(path string, config *grok.Config) error {
	if config.Patterns == nil {
		config.Patterns = make(map[string]string)
	}

	// Check if the path points to a directory. If so, append the glob pattern to match .grok files
	if fi, err := os.Stat(path); err == nil {
		if fi.IsDir() {
			path = filepath.Join(path, "*.grok")
		}
	} else {
		return fmt.Errorf("invalid path: %s", path)
	}

	files, err := filepath.Glob(path)
	if err != nil {
		return err
	}

	for _, fileName := range files {
		file, err := os.Open(fileName)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 || line[0] == '#' {
				// Skip empty lines or lines starting with a hash (comments)
				continue
			}

			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				// Skip lines that don't seem to fit the expected pattern
				continue
			}

			key := parts[0]
			pattern := parts[1]
			config.Patterns[key] = pattern
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}
	return nil
}

func startSyslogServer(grok *grok.Grok, config *grok.Config, publisher Publisher) {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	log.Printf("Booting syslog server")
	server.ListenUDP("0.0.0.0:1514")
	server.Boot()

	processor := &SyslogProcessor{
		Grok:      grok,
		Config:    config,
		Publisher: publisher,
	}

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			go processor.ProcessLogMessage(logParts)
		}
	}(channel)

	server.Wait()
}

// Convert syslog metadata fields into tags
func extractSyslogMetadataAsTags(logParts map[string]interface{}) map[string]string {
	tags := make(map[string]string)
	for k, v := range logParts {
		if k != "message" { // Skip the actual message content
			tags[k] = fmt.Sprintf("%v", v) // Convert the value to string format
		}
	}
	return tags
}
