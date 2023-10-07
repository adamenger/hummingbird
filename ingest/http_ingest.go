package main

import (
  "fmt"
  "net/http"
  "encoding/json"
)

func httpIngestHandler(w http.ResponseWriter, r *http.Request) {
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


