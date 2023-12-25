package slogkafka

import (
	"context"
	"encoding/json"
	"time"

	"log/slog"

	slogcommon "github.com/samber/slog-common"
	"github.com/segmentio/kafka-go"
)

type Option struct {
	// log level (default: debug)
	Level slog.Leveler

	// Kafka Writer
	KafkaWriter *kafka.Writer

	// optional: customize Kafka event builder
	Converter Converter

	// optional: see slog.HandlerOptions
	AddSource   bool
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
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

var _ slog.Handler = (*KafkaHandler)(nil)

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

	payload := converter(h.option.AddSource, h.option.ReplaceAttr, h.attrs, h.groups, &record)

	return h.publish(ctx, record.Time, payload)
}

func (h *KafkaHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &KafkaHandler{
		option: h.option,
		attrs:  slogcommon.AppendAttrsToGroup(h.groups, h.attrs, attrs...),
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

func (h *KafkaHandler) publish(ctx context.Context, timestamp time.Time, payload map[string]interface{}) error {
	key, err := timestamp.MarshalBinary()
	if err != nil {
		return err
	}

	values, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return h.option.KafkaWriter.WriteMessages(
		ctx,
		kafka.Message{
			Key:   key,
			Value: values,
		},
	)
}
