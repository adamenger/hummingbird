package publisher

import (
	"github.com/IBM/sarama"
)

type KafkaPublisher struct {
	Broker string
	Topic  string
}

func NewKafkaPublisher(broker string, topic string) *KafkaPublisher {
	return &KafkaPublisher{
		Broker: broker,
		Topic:  topic,
	}
}


func (kp *KafkaPublisher) Publish(message []byte) error {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{kp.Broker}, config)
	if err != nil {
		return err
	}
	defer producer.Close()

	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: kp.Topic,
		Value: sarama.ByteEncoder(message),
	})

	return err
}
