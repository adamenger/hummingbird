package main

import (
	"log"
	"net/http"

	"github.com/trivago/grok"
)

const (
	broker = "localhost:29092"
	topic  = "log_topic"
)

type LogData struct {
	Tags          map[string]string      `json:"tags"`
	Message       string                 `json:"message"`
	ParsedMessage map[string]interface{} `json:"parsedMessage,omitempty"` // will be omitted from JSON if empty
}

func main() {
	config := grok.Config{}
	err := loadPatterns("patterns/", &config)
	if err != nil {
		log.Printf("Failed to load grok patterns: %s", err)
	}

	g, err := grok.New(config)
	if err != nil {
		log.Printf("Failed to boot grok parser: %s", err)
	}

	kp := &KafkaPublisher{}
	go startSyslogServer(g, &config, kp) // start syslog server

	filebeatProcessor := &FilebeatProcessor{
		Grok:      g,
		Publisher: kp,
	}

	httpProcessor := &HttpProcessor{
		Grok:      g,
		Publisher: kp,
	}

	http.HandleFunc("/ingest", httpProcessor.HandleRequest)
	http.HandleFunc("/filebeat", filebeatProcessor.HandleRequest)
	log.Fatal(http.ListenAndServe(":8089", nil))
}
