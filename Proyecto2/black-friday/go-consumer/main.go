package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

// Estructura de la venta recibida desde Kafka
type Venta struct {
	Categoria       int32   `json:"categoria"`
	ProductoID      string  `json:"producto_id"`
	Precio          float64 `json:"precio"`
	CantidadVendida int32   `json:"cantidad_vendida"`
}

// Mapeo de categorías
var categorias = map[int32]string{
	1: "Electronica",
	2: "Ropa",
	3: "Hogar",
	4: "Belleza",
}

// Función principal del consumidor
func main() {
	// Configuración desde variables de entorno
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

	// Verificar conexión a Valkey
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Error al conectar a Valkey en %s: %v", valkeyAddr, err)
	}

	// Configuración del consumidor de Kafka
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	master, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Fatalf("Error creando Kafka: %v", err)
	}
	defer master.Close()

	// Crear consumidor de la partición
	consumer, err := master.ConsumePartition("sales-topic", 0, sarama.OffsetOldest)
	if err != nil {
		log.Fatalf("Error conectando a la partición: %v", err)
	}
	defer consumer.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// Bucle principal para procesar mensajes
	for {
		select {
		case err := <-consumer.Errors():
			log.Printf("Error en Kafka: %v", err)
		case msg := <-consumer.Messages():
			procesarMensaje(ctx, rdb, msg.Value)
		case <-signals:
			return
		}
	}
}

// Procesa cada mensaje recibido de Kafka
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

	keyContador := fmt.Sprintf("contador:%s", nombreCat)
	rdb.Incr(ctx, keyContador)

	// Actualizar ranking de productos
	keyRanking := fmt.Sprintf("ranking:%s", nombreCat)
	rdb.ZIncrBy(ctx, keyRanking, float64(venta.CantidadVendida), venta.ProductoID)

	datosGrafica := map[string]interface{}{
		"producto": venta.ProductoID,
		"precio":   venta.Precio,
		"ts":       time.Now().UnixMilli(),
	}
	// Guardar datos para la gráfica
	jsonGrafica, _ := json.Marshal(datosGrafica)
	keyHistorial := fmt.Sprintf("historial:%s", nombreCat)
	rdb.LPush(ctx, keyHistorial, jsonGrafica)

	log.Printf("[Guardado] Cat: %s | Prod: %s | Cant: %d",
		nombreCat, venta.ProductoID, venta.CantidadVendida)
}