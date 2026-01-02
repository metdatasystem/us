package streaming

import (
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AWIPSRaw struct {
	Issued time.Time `json:"issued"`
	TTAAII string    `json:"ttaaii"`
	CCCC   string    `json:"cccc"`
	AWIPS  string    `json:"awips"`
	Text   string    `json:"text"`
}

func (a *AWIPSRaw) Marshal() ([]byte, error) {
	return json.Marshal(a)
}

func (a *AWIPSRaw) Unmarshal(data []byte) error {
	return json.Unmarshal(data, a)
}

func DeclareAWIPSQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"awips.queue", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
}
