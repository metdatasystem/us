package main

import (
	"fmt"
	"os"

	"github.com/metdatasystem/us/pkg/streaming"
	amqp "github.com/rabbitmq/amqp091-go"
)

func newRabbitChannel() (*amqp.Channel, error) {
	conn, err := amqp.Dial(os.Getenv("RABBIT_URL"))
	if err != nil {
		return nil, err
	}

	return conn.Channel()

}

func initRabbit(ch *amqp.Channel) error {
	var err error

	err = streaming.DeclareLiveExchange(ch)
	if err != nil {
		return fmt.Errorf("failed to declare %s", streaming.ExchangeLiveName)
	}

	return nil
}
