package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings" // <--- IMPORTANTE: Necesario para procesar la lista de brokers
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

// Estructura de la venta (Mismo JSON que envía Locust/Rust)
type Venta struct {
	Categoria       int32   `json:"categoria"`
	ProductoID      string  `json:"producto_id"`
	Precio          float64 `json:"precio"`
	CantidadVendida int32   `json:"cantidad_vendida"`
}

// Mapa de categorías
var categorias = map[int32]string{
	1: "Electronica",
	2: "Ropa",
	3: "Hogar",
	4: "Belleza",
}

func main() {
	// --- LÓGICA DE CONEXIÓN DINÁMICA ---
	// 1. Configurar Valkey (Redis)
	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "localhost:6379" // Default local
	}

	// 2. Configurar Kafka
	kafkaEnv := os.Getenv("KAFKA_BROKERS")
	if kafkaEnv == "" {
		kafkaEnv = "localhost:9092" // Default local
	}
	brokers := strings.Split(kafkaEnv, ",")

	log.Printf("🔌 Consumer config -> Valkey: %s | Kafka: %v", valkeyAddr, brokers)
	// -----------------------------------

	// Conexión a Valkey usando la variable
	rdb := redis.NewClient(&redis.Options{
		Addr: valkeyAddr,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Error crítico: No se puede conectar a Valkey en %s: %v", valkeyAddr, err)
	}
	log.Println("✅ Conectado a Valkey correctamente.")

	// Configuración de Kafka
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	// Usamos la variable 'brokers'
	master, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Fatalf("❌ Error creando consumidor Kafka: %v", err)
	}
	defer master.Close()

	// Suscribirse al tópico
	consumer, err := master.ConsumePartition("sales-topic", 0, sarama.OffsetOldest)
	if err != nil {
		log.Fatalf("❌ Error conectando a la partición: %v", err)
	}
	defer consumer.Close()

	// Manejar Ctrl+C
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	log.Println("🚀 Consumidor Iniciado. Esperando ventas de Kafka...")

	// Bucle de procesamiento
	for {
		select {
		case err := <-consumer.Errors():
			log.Printf("⚠️ Error en Kafka: %v", err)
		case msg := <-consumer.Messages():
			procesarMensaje(ctx, rdb, msg.Value)
		case <-signals:
			log.Println("🛑 Cerrando consumidor...")
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

	// --- LÓGICA PARA GRAFANA ---

	// 1. Contador total
	keyContador := fmt.Sprintf("contador:%s", nombreCat)
	rdb.Incr(ctx, keyContador)

	// 2. Ranking producto más vendido
	keyRanking := fmt.Sprintf("ranking:%s", nombreCat)
	rdb.ZIncrBy(ctx, keyRanking, float64(venta.CantidadVendida), venta.ProductoID)

	// 3. Historial de precios
	datosGrafica := map[string]interface{}{
		"producto": venta.ProductoID,
		"precio":   venta.Precio,
		"ts":       time.Now().UnixMilli(),
	}
	jsonGrafica, _ := json.Marshal(datosGrafica)
	keyHistorial := fmt.Sprintf("historial:%s", nombreCat)
	rdb.LPush(ctx, keyHistorial, jsonGrafica)

	log.Printf("💾 [Guardado en Valkey] Cat: %s | Prod: %s | Cant: %d",
		nombreCat, venta.ProductoID, venta.CantidadVendida)
}