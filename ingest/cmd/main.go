package main

import (
	"log"
	"net/http"
  "github.com/adamenger/hummingbird/ingest/publisher"
  "github.com/adamenger/hummingbird/ingest/processor"
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
	
  kp := &publisher.KafkaPublisher{
    Broker:  "localhost:9092",
    Topic:   "filebeat",
  }
	syslogProcessor, err := processor.NewSyslogProcessor("patterns/syslog", kp)
	if err != nil {
		log.Fatalf("Failed to initialize syslog processor: %v", err)
	}

	go syslogProcessor.StartSyslogServer() // start syslog server
	if err != nil {
		log.Fatalf("Failed to boot syslog server: %v", err)
	}

	filebeatProcessor := &processor.FilebeatProcessor{
		//Grok:      syslogProcessor.Grok,
		Publisher: &publisher.KafkaPublisher{
      Broker:  "localhost:9092",
      Topic:   "filebeat",
    },
	}

	httpProcessor := &processor.HttpProcessor{
		//Grok:      syslogProcessor.Grok,
		Publisher: &publisher.KafkaPublisher{
      Broker:  "localhost:9092",
      Topic:   "http",
    },
	}

	http.HandleFunc("/ingest", httpProcessor.HandleRequest)
	http.HandleFunc("/filebeat", filebeatProcessor.HandleRequest)
	log.Fatal(http.ListenAndServe(":8089", nil))
}
