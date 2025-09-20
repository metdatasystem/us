package internal

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var producer *Producer
var t = time.Now()

func Local(path string, loglevel zerolog.Level) {
	var err error
	producer, err = NewProducer()
	if err != nil {
		log.Error().Err(err).Msg("failed to create producer")
		return
	}

	process(path)

	producer.Stop()

}

func process(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Error().Err(err).Msg("failed to open file")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Error().Err(err).Msg("failed to stat file")
		return
	}

	if stat.IsDir() {
		files, err := file.ReadDir(0)
		if err != nil {
			log.Error().Err(err).Msg("failed to read directory")
			return
		}

		for _, f := range files {
			process(f.Name())
		}
	} else {
		processFile(file, stat.Size())
	}
}

func processFile(file *os.File, size int64) {
	data := make([]byte, size)
	_, err := file.Read(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to read file")
	}

	if producer == nil {
		log.Error().Err(err).Msg("producer is nil")
		return
	}
	message := producer.NewMessage(string(data), t)
	err = producer.SendMessage(message)
	if err != nil {
		log.Error().Err(err).Msg("failed to send message")
	}
}
