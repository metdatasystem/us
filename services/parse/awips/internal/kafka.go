package internal

import (
	"os"

	"github.com/twmb/franz-go/pkg/kgo"
)

func newKafkaClient() (*kgo.Client, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(os.Getenv("KAFKA_BROKERS")),
		kgo.ConsumerGroup("us-awips-raw"),
		kgo.ConsumeTopics("us-awips-raw"),
		kgo.DisableAutoCommit(),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}
