# Go commands
GO=go
GOFMT=gofmt
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOTEST=$(GO) test

# Binaries output directory
OUTPUT_DIR=bin

# Targets
all: ingest digest

ingest:
	cd ingest/cmd && $(GOBUILD) -o ../../$(OUTPUT_DIR)/hummingbird_ingest -v

digest:
	cd digest/cmd && $(GOBUILD) -o ../$(OUTPUT_DIR)/hummingbird_digest -v

format:
	$(GOFMT) -s -w ingest/.
	$(GOFMT) -s -w digest/.

clean:
	$(GOCLEAN)
	rm -f $(OUTPUT_DIR)/hummingbird_ingest
	rm -f $(OUTPUT_DIR)/hummingbird_digest

test:
	cd ingest && $(GOTEST) -v ./...
	cd digest && $(GOTEST) -v ./...

run-ingest: ingest
	./$(OUTPUT_DIR)/hummingbird_ingest

run-digest: digest
	./$(OUTPUT_DIR)/hummingbird_digest

.PHONY: all ingest digest format clean test run-ingest run-digest
