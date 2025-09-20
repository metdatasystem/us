package internal

import (
	"log/slog"
	"os"
	"time"
)

var producer *Producer
var t = time.Now()

func Local(path string) {
	var err error
	producer, err = NewProducer()
	if err != nil {
		slog.Error("failed to create producer", "error", err.Error())
	}

	process(path)

	producer.Stop()

}

func process(path string) {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("failed to open file", "error", err.Error())
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		slog.Error("failed to stat file", "error", err.Error())
		return
	}

	if stat.IsDir() {
		files, err := file.ReadDir(0)
		if err != nil {
			slog.Error("failed to read directory", "error", err.Error())
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
		slog.Error("failed to read file", "error", err.Error())
	}

	if producer == nil {
		slog.Error("producer is nil")
		return
	}
	message := producer.NewMessage(string(data), t)
	err = producer.SendMessage(message)
	if err != nil {
		slog.Error("failed to send message", "error", err.Error())
	}
}
