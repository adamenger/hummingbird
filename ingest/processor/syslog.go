package processor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mcuadros/go-syslog"
	"github.com/trivago/grok"
  "github.com/adamenger/hummingbird/ingest"
  "github.com/adamenger/hummingbird/ingest/publisher"
)

type SyslogProcessor struct {
	Grok          *grok.Grok
	Config        *grok.Config
	Publisher     publisher.Publisher
	PatternsPath  string
	SyslogChannel syslog.LogPartsChannel
}

func NewSyslogProcessor(patternsPath string, publisher publisher.Publisher) (*SyslogProcessor, error) {
  grokConfig := grok.Config{}
  syslogGrok, err := grok.New(grokConfig)
	if err != nil {
		return nil, err
	}

	sp := &SyslogProcessor{
    Grok:          syslogGrok,
		PatternsPath:  patternsPath,
		SyslogChannel: make(syslog.LogPartsChannel),
    Publisher:     publisher,
	}
	
  err = sp.LoadPatterns()
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func (sp *SyslogProcessor) StartSyslogServer() error {
	handler := syslog.NewChannelHandler(sp.SyslogChannel)

	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:1514")
	server.Boot()
	log.Printf("Booted syslog server")

	go func(channel syslog.LogPartsChannel) {
		for logParts := range sp.SyslogChannel {
			go sp.ProcessLogMessage(logParts)
		}
	}(sp.SyslogChannel)

	server.Wait()

	return nil
}

func (sp *SyslogProcessor) LoadPatterns() error {
	if sp.Config.Patterns == nil {
		sp.Config.Patterns = make(map[string]string)
	}

	// Check if the path points to a directory. If so, append the glob pattern to match .grok files
	if fi, err := os.Stat(sp.PatternsPath); err == nil {
		if fi.IsDir() {
			sp.PatternsPath = filepath.Join(sp.PatternsPath, "*.grok")
		}
	} else {
		return fmt.Errorf("invalid path: %s", sp.PatternsPath)
	}

	files, err := filepath.Glob(sp.PatternsPath)
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
			sp.Config.Patterns[key] = pattern
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}
	return nil

}

func (sp *SyslogProcessor) ProcessLogMessage(logParts map[string]interface{}) {
	var message string
	if contentVal, ok := logParts["message"]; ok && contentVal != nil {
		message = contentVal.(string)
	} else {
		log.Println("No content in logParts or content is nil")
		return
	}

	tags := extractSyslogMetadataAsTags(logParts)
	parsedMessage := map[string]interface{}{}

	// Before Grok parsing
	if len(message) > 0 {
		// Loop through patterns
		for _, pattern := range sp.Config.Patterns {
			values, err := sp.Grok.Parse(pattern, []byte(message))
			if err == nil && len(values) > 0 {
				for k, v := range values {
					parsedMessage[k] = string(v)
				}
			}
		}
	} else {
		log.Println("No message in logParts or message is nil")
	}

	// Convert the structured message to JSON format for Kafka
	logData := ingest.LogData{
		Tags:          tags,
		Message:       message,
		ParsedMessage: parsedMessage,
	}

	jsonData, err := json.Marshal(logData)
	if err != nil {
		log.Printf("Error marshaling log data: %v", err)
		return
	}

	err = sp.Publisher.Publish(jsonData)
	if err != nil {
		log.Printf("Failed to produce message to Kafka: %v", err)
	}
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
