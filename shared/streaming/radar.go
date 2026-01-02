package streaming

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func DeclareNEXRADSqsQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"nexrad.queue", // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
}
