package internal

import (
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Local(path string, logLevel zerolog.Level) {
	zerolog.SetGlobalLevel(logLevel)

	db, err := newDatabasePool(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise database")
		return
	}

	rabbitChannel, err := newRabbitChannel()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise rabbit channel")
		return
	}

	err = initRabbit(rabbitChannel)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise rabbit declarations")
		return
	}

	process(path, db, rabbitChannel)

}

func process(path string, db *pgxpool.Pool, rabbit *amqp.Channel) {
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
			process(path+f.Name(), db, rabbit)
		}
	} else {
		processFile(file, stat.Size(), db, rabbit)
	}
}

func processFile(file *os.File, size int64, db *pgxpool.Pool, rabbit *amqp.Channel) {
	data := make([]byte, size)
	_, err := file.Read(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to read file")
	}

	text := string(data)

	HandleText(text, time.Now(), db, rabbit)
	time.Sleep(10 * time.Second)
}
