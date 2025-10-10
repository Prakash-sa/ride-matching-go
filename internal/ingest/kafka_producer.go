package ingest

import (
	"context"
	"encoding/json"
	"time"

	"github.com/example/ride-matching/internal/models"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	w := kafka.NewWriter(kafka.WriterConfig{Brokers: brokers, Topic: topic, Balancer: &kafka.LeastBytes{}})
	return &KafkaProducer{writer: w}
}

func (k *KafkaProducer) PublishLocation(d models.Driver) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	b, _ := json.Marshal(d)
	return k.writer.WriteMessages(ctx, kafka.Message{Key: []byte(d.ID), Value: b})
}

func (k *KafkaProducer) Close() error {
	if k.writer == nil {
		return nil
	}
	return k.writer.Close()
}
