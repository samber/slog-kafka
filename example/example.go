package main

import (
	"context"
	"fmt"
	"time"

	slogkafka "github.com/samber/slog-kafka/v2"
	"github.com/segmentio/kafka-go"

	"log/slog"
)

func main() {
	// docker-compose up -d

	uri := "127.0.0.1:9092"

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", uri)
	if err != nil {
		panic(err)
	}

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             "logs",
		NumPartitions:     12,
		ReplicationFactor: 1,
	})
	if err != nil {
		panic(err)
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{uri},
		Topic:   "logs",
		Dialer:  dialer,

		Async:       true,
		Balancer:    &kafka.Hash{},
		MaxAttempts: 3,

		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			fmt.Printf(msg+"\n", args...)
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			fmt.Printf(msg+"\n", args...)
		}),
	})

	defer writer.Close()
	defer conn.Close()

	logger := slog.New(slogkafka.Option{Level: slog.LevelDebug, KafkaWriter: writer}.NewKafkaHandler())
	logger = logger.With("release", "v1.0.0")

	logger.
		With(
			slog.Group("user",
				slog.String("id", "user-123"),
				slog.Time("created_at", time.Now()),
			),
		).
		With("error", fmt.Errorf("an error")).
		Error("a message")
}
