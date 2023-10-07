package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/trivago/grok"
)

type FilebeatHandler struct {
	Grok *grok.Grok
}

func (f *FilebeatHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Ensure it's a POST request
	if r.Method != "POST" {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Assuming the Filebeat payload is a nested JSON
	// with metadata and a message field
	var payload map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&payload)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	// Convert metadata to tags and populate the LogData struct
	var data LogData
	data.Tags = make(map[string]string)
	for k, v := range payload {
		if k != "message" {
			data.Tags[k] = fmt.Sprintf("%v", v)
		} else {
			data.Message = fmt.Sprintf("%v", v)
		}
	}

	// Check if the parser tag is available
	parser, ok := data.Tags["parser"]
	if ok {
		values, err := f.Grok.Parse(parser, []byte(data.Message))
		if err == nil {
			for k, v := range values {
				data.ParsedMessage[k] = v
			}
		}
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
