package main

import (
	"log"
	"net/http"
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
	kp := &KafkaPublisher{}

	syslogProcessor, err := NewSyslogProcessor("patterns/")
	if err != nil {
		log.Fatalf("Failed to initialize syslog processor: %v", err)
	}

	go syslogProcessor.StartSyslogServer() // start syslog server
	if err != nil {
		log.Fatalf("Failed to boot syslog server: %v", err)
	}

	filebeatProcessor := &FilebeatProcessor{
		Grok:      syslogProcessor.Grok,
		Publisher: kp,
	}

	httpProcessor := &HttpProcessor{
		Grok:      syslogProcessor.Grok,
		Publisher: kp,
	}

	http.HandleFunc("/ingest", httpProcessor.HandleRequest)
	http.HandleFunc("/filebeat", filebeatProcessor.HandleRequest)
	log.Fatal(http.ListenAndServe(":8089", nil))
}
