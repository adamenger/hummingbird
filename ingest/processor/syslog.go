package processor

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/mcuadros/go-syslog"
  "github.com/adamenger/hummingbird/ingest"
  "github.com/adamenger/hummingbird/ingest/publisher"
  "github.com/adamenger/hummingbird/ingest/parser"
)

type SyslogProcessor struct {
	GrokParser    *parser.GrokParser
	Publisher     publisher.Publisher
	SyslogChannel syslog.LogPartsChannel
}

func NewSyslogProcessor(patternsPath string, publisher publisher.Publisher) (*SyslogProcessor, error) {
  gp, err := parser.NewGrokParser(patternsPath)
	if err != nil {
		return nil, err
	}

	sp := &SyslogProcessor{
    GrokParser:    gp,
    Publisher:     publisher,
		SyslogChannel: make(syslog.LogPartsChannel),
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
		for _, pattern := range sp.GrokParser.Patterns {
			values, err := sp.GrokParser.Grok.Parse(pattern, []byte(message))
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
    log.Printf("syslog: Failed to produce message to Kafka: %v", err)
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
