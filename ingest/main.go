package main

import (
	"log"
	"net/http"

	"github.com/IBM/sarama"
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

func produceToKafka(message []byte) error {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{broker}, config)
	if err != nil {
		return err
	}
	defer producer.Close()

	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(message),
	})

	return err
}

func main() {
	config := grok.Config{}
	err := loadPatterns("patterns/", &config)
	if err != nil {
		log.Printf("Failed to load grok patterns", err)
	}

	g, err := grok.New(config)
	if err != nil {
		log.Printf("Failed to boot grok parser", err)
	}
	go startSyslogServer(g, &config) // Start syslog server

	// set up filebeatHandler and pass grok object
	filebeatHandler := &FilebeatHandler{
		Grok: g,
	}

	http.HandleFunc("/ingest", httpIngestHandler)
	http.HandleFunc("/filebeat", filebeatHandler.HandleRequest)
	log.Fatal(http.ListenAndServe(":8089", nil))
}
