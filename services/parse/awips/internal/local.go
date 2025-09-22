package internal

import (
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

func Local(path string) {
	db, err := newDatabasePool()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise database")
		return
	}

	kafkaClient, err := newKafkaClient()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise kafka client")
		return
	}

	process(path, db, kafkaClient)

}

func process(path string, db *pgxpool.Pool, kafka *kgo.Client) {
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
			process(f.Name(), db, kafka)
		}
	} else {
		processFile(file, stat.Size(), db, kafka)
	}
}

func processFile(file *os.File, size int64, db *pgxpool.Pool, kafka *kgo.Client) {
	data := make([]byte, size)
	_, err := file.Read(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to read file")
	}

	text := string(data)

	Handle(text, time.Now(), db, kafka)
}
