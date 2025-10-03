package streaming

import amqp "github.com/rabbitmq/amqp091-go"

const ExchangeLiveName = "live.exchange"
const ExchangeLiveType = "direct"

func DeclareLiveExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		ExchangeLiveName, // name
		ExchangeLiveType, // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
}
