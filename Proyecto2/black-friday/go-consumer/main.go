package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

var (
	ventasTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "blackfriday_ventas_total",
			Help: "Numero total de ventas procesadas por categoria",
		},
		[]string{"categoria"},
	)

	ingresosTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "blackfriday_ingresos_total",
			Help: "Suma total de dinero vendido por categoria",
		},
		[]string{"categoria"},
	)
)

func init() {
	prometheus.MustRegister(ventasTotal)
	prometheus.MustRegister(ingresosTotal)
}

func main() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Servidor de metricas iniciado en :2112")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Error iniciando servidor HTTP: %v", err)
		}
	}()

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
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Error al conectar a Valkey: %v", err)
	}

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	master, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Fatalf("Error creando Kafka: %v", err)
	}
	defer master.Close()

	consumer, err := master.ConsumePartition("sales-topic", 0, sarama.OffsetOldest)
	if err != nil {
		log.Fatalf("Error conectando a partición: %v", err)
	}
	defer consumer.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	log.Println("Consumidor iniciado. Esperando mensajes...")

	for {
		select {
		case err := <-consumer.Errors():
			log.Printf("Error Kafka: %v", err)
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
		log.Printf("Error JSON: %v", err)
		return
	}

	nombreCat, existe := categorias[venta.Categoria]
	if !existe {
		nombreCat = "Otros"
	}

	ventasTotal.WithLabelValues(nombreCat).Inc()
	ingresosTotal.WithLabelValues(nombreCat).Add(venta.Precio)

	keyContador := fmt.Sprintf("contador:%s", nombreCat)
	rdb.Incr(ctx, keyContador)

	rdb.ZIncrBy(ctx, fmt.Sprintf("ranking:%s", nombreCat), float64(venta.CantidadVendida), venta.ProductoID)
	rdb.ZIncrBy(ctx, "ranking:general", float64(venta.CantidadVendida), venta.ProductoID)

	rdb.ScriptLoad(ctx, `
        local current = tonumber(redis.call('get', KEYS[1]) or 0)
        if tonumber(ARGV[1]) > current then
            redis.call('set', KEYS[1], ARGV[1])
        end
    `)

	currentMax, _ := rdb.Get(ctx, "precio:max:general").Float64()
	if venta.Precio > currentMax {
		rdb.Set(ctx, "precio:max:general", venta.Precio, 0)
	}

	currentMin, err := rdb.Get(ctx, "precio:min:general").Float64()
	if err == redis.Nil || venta.Precio < currentMin {
		rdb.Set(ctx, "precio:min:general", venta.Precio, 0)
	}

	keySuma := fmt.Sprintf("suma_precio:%s", nombreCat)
	nuevaSuma, _ := rdb.IncrByFloat(ctx, keySuma, venta.Precio).Result()

	contador, _ := rdb.Get(ctx, keyContador).Int64()

	if contador > 0 {
		promedio := nuevaSuma / float64(contador)
		rdb.Set(ctx, fmt.Sprintf("promedio:precio:%s", nombreCat), promedio, 0)
	}

	if venta.Categoria == 4 {
		rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: "stream:belleza",
			Values: map[string]interface{}{
				"precio":   venta.Precio,
				"producto": venta.ProductoID,
			},
		})
	}

	log.Printf("[Procesado] Cat: %s | Prod: %s | $%.2f", nombreCat, venta.ProductoID, venta.Precio)
}