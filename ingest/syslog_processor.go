package main

import (
	"encoding/json"
	"log"

	"github.com/trivago/grok"
)

type SyslogProcessor struct {
	Grok      *grok.Grok
	Config    *grok.Config
	Publisher Publisher
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
	logData := LogData{
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
