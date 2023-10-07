package main

import (
	"encoding/json"
	"fmt"
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

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure it's a POST request
	if r.Method != "POST" {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Parse the request body into the LogData struct
	var data LogData
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	// Check if required fields (tags and message) are present and valid
	if len(data.Tags) == 0 || data.Message == "" {
		http.Error(w, "Invalid data format", http.StatusBadRequest)
		return
	}

	// Convert the struct back to JSON for Kafka
	message, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal data", http.StatusInternalServerError)
		return
	}

	// Produce the message to Kafka
	err = produceToKafka(message)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to produce message to Kafka: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Message ingested successfully"))
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

	http.HandleFunc("/ingest", ingestHandler)
	http.HandleFunc("/filebeat", filebeatHandler.HandleRequest)
	log.Fatal(http.ListenAndServe(":8089", nil))
}
