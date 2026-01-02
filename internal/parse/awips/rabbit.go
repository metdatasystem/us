package internal

import (
	"fmt"
	"os"

	"github.com/metdatasystem/us/shared/streaming"
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

	q, err := streaming.DeclareAWIPSQueue(ch)
	if err != nil {
		return fmt.Errorf("failed to declare %s", q.Name)
	}

	err = streaming.DeclareLiveExchange(ch)
	if err != nil {
		return fmt.Errorf("failed to declare %s", streaming.ExchangeLiveName)
	}

	return nil
}
