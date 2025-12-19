package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

type Venta struct {
	Categoria       int32   `json:"categoria"`
	ProductoID      string  `json:"producto_id"`
	Precio          float64 `json:"precio"`
	CantidadVendida int32   `json:"cantidad_vendida"`
}

var categorias = map[int32]string{
	1: "Electronica",
	2: "Ropa",
	3: "Hogar",
	4: "Belleza",
}

func main() {
	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "localhost:6379"
	}

	kafkaEnv := os.Getenv("KAFKA_BROKERS")
	if kafkaEnv == "" {
		kafkaEnv = "localhost:9092"
	}
	brokers := strings.Split(kafkaEnv, ",")

	rdb := redis.NewClient(&redis.Options{
		Addr: valkeyAddr,
	})

	ctx := context.Background()
	
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	master, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Fatalf("Fatal Kafka: %v", err)
	}
	defer master.Close()

	consumer, err := master.ConsumePartition("sales-topic", 0, sarama.OffsetOldest)
	if err != nil {
		log.Fatalf("Fatal Partition: %v", err)
	}
	defer consumer.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	for {
		select {
		case err := <-consumer.Errors():
			log.Printf("Error: %v", err)
		case msg := <-consumer.Messages():
			procesarMensaje(ctx, rdb, msg.Value)
		case <-signals:
			return
		}
	}
}

func procesarMensaje(ctx context.Context, rdb *redis.Client, value []byte) {
	var venta Venta
	if err := json.Unmarshal(value, &venta); err != nil {
		return
	}

	nombreCat, existe := categorias[venta.Categoria]
	if !existe {
		nombreCat = "Otros"
	}

	keyContador := fmt.Sprintf("contador:%s", nombreCat)
	rdb.Incr(ctx, keyContador)
	rdb.Incr(ctx, "total_ventas")
	rdb.ZIncrBy(ctx, "ranking_productos", float64(venta.CantidadVendida), venta.ProductoID)
}