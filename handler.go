package slogkafka

import (
	"context"
	"encoding/json"
	"time"

	"log/slog"

	"github.com/segmentio/kafka-go"
)

type Option struct {
	// log level (default: debug)
	Level slog.Leveler

	// Kafka Writer
	KafkaWriter *kafka.Writer

	// optional: customize Kafka event builder
	Converter Converter
}

func (o Option) NewKafkaHandler() slog.Handler {
	if o.Level == nil {
		o.Level = slog.LevelDebug
	}

	if o.KafkaWriter == nil {
		panic("missing Kafka writer")
	}

	return &KafkaHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

type KafkaHandler struct {
	option Option
	attrs  []slog.Attr
	groups []string
}

func (h *KafkaHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *KafkaHandler) Handle(ctx context.Context, record slog.Record) error {
	converter := DefaultConverter
	if h.option.Converter != nil {
		converter = h.option.Converter
	}

	payload := converter(h.attrs, record)

	return h.publish(record.Time, payload)
}

func (h *KafkaHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &KafkaHandler{
		option: h.option,
		attrs:  appendAttrsToGroup(h.groups, h.attrs, attrs),
		groups: h.groups,
	}
}

func (h *KafkaHandler) WithGroup(name string) slog.Handler {
	return &KafkaHandler{
		option: h.option,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

func (h *KafkaHandler) publish(timestamp time.Time, payload map[string]interface{}) error {
	key, err := timestamp.MarshalBinary()
	if err != nil {
		return err
	}

	values, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return h.option.KafkaWriter.WriteMessages(
		context.Background(),
		kafka.Message{
			Key:   key,
			Value: values,
		},
	)
}
