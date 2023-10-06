package main

import (
  "log"
  "fmt"
  "encoding/json"
  "os"
  "path/filepath"
  "bufio"
  "strings"

  "github.com/trivago/grok"
  "github.com/mcuadros/go-syslog"
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

func startSyslogServer(grok *grok.Grok, config *grok.Config) {
  channel := make(syslog.LogPartsChannel)
  handler := syslog.NewChannelHandler(channel)

  server := syslog.NewServer()
  server.SetFormat(syslog.Automatic)
  server.SetHandler(handler)
  log.Printf("Booting syslog server")
  server.ListenUDP("0.0.0.0:1514")
  server.Boot()

  go func(channel syslog.LogPartsChannel) {
    for logParts := range channel {
      var message string
      if contentVal, ok := logParts["message"]; ok && contentVal != nil {
        message = contentVal.(string)
      } else {
        log.Println("No content in logParts or content is nil")
        continue // skip this iteration
      }

      tags := extractSyslogMetadataAsTags(logParts)
      parsedMessage := map[string]interface{}{}

      // Before Grok parsing
      if len(message) > 0  {
        // Loop through patterns
        for _, pattern := range config.Patterns {
          values, err := grok.Parse(pattern, []byte(message))
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
        continue
      }

      err = produceToKafka(jsonData)
      if err != nil {
        log.Printf("Failed to produce message to Kafka: %v", err)
      }

      log.Printf("Sent message to kafka")
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
