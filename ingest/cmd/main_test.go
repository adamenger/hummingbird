package main

import (
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Setup
	err := exec.Command("docker-compose", "up", "-d").Run()
	if err != nil {
		log.Fatalf("Failed to start docker-compose: %v", err)
	}
	time.Sleep(10 * time.Second) // give Kafka some time to fully start

	// Run tests
	exitVal := m.Run()

	// Teardown
	exec.Command("docker-compose", "down").Run()
	if err != nil {
		log.Fatalf("Failed to stop docker-compose: %v", err)
	}
	time.Sleep(5 * time.Second) // allow time for graceful shutdown

	// Exit with test run result
	os.Exit(exitVal)
}

func TestKafkaIntegration(t *testing.T) {
	// Given: A KafkaPublisher
	kp := &KafkaPublisher{}

	// And: A test message
	message := []byte("Sample message for integration testing")

	// When: We publish the message to Kafka
	err := kp.Publish(message)
	assert.Nil(t, err, "Expected no error when publishing message")

	// Then: We should be able to consume that message from Kafka and validate its content
	consumer, err := sarama.NewConsumer([]string{broker}, nil)
	assert.Nil(t, err, "Failed to create consumer")
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	assert.Nil(t, err, "Failed to create partition consumer")
	defer partitionConsumer.Close()

	select {
	case msg := <-partitionConsumer.Messages():
		assert.Equal(t, message, msg.Value, "The consumed message value is not the same as the published value")
	case <-time.After(10 * time.Second): // adjust timeout as needed
		t.Fatal("Failed to consume the message within the expected time")
	}
}
