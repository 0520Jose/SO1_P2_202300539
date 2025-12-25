package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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
	1: "Electronica", 2: "Ropa", 3: "Hogar", 4: "Belleza",
}

// Consumer representa el consumidor del grupo
type Consumer struct {
	rdb *redis.Client
}

// Setup se ejecuta al inicio de una nueva sesión (no lo usaremos por ahora)
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error { return nil }

// Cleanup se ejecuta al final de una sesión
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

// ConsumeClaim es el bucle principal que lee los mensajes
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		// Procesamos el mensaje
		procesarMensaje(session.Context(), consumer.rdb, message.Value)
		
		// Marcamos el mensaje como procesado (commit)
		session.MarkMessage(message, "")
	}
	return nil
}

func main() {
	// Configuración de Redis y Kafka (Igual que antes)
	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" { valkeyAddr = "localhost:6379" }
	
	kafkaEnv := os.Getenv("KAFKA_BROKERS")
	if kafkaEnv == "" { kafkaEnv = "localhost:9092" }
	brokers := strings.Split(kafkaEnv, ",")

	rdb := redis.NewClient(&redis.Options{Addr: valkeyAddr})

	// Configuración del Consumer Group
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
    // Importante: Balanceo de carga entre consumidores
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()

	// Creamos el cliente de Grupo
	groupName := "black-friday-group" // Todos los pods compartirán este ID
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupName, config)
	if err != nil {
		log.Fatalf("Error creando consumer group: %v", err)
	}
	defer consumerGroup.Close()

	// Contexto para cancelar
	ctx, cancel := context.WithCancel(context.Background())
	consumer := &Consumer{rdb: rdb}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Bucle para consumir
	go func() {
		defer wg.Done()
		for {
			if err := consumerGroup.Consume(ctx, []string{"sales-topic"}, consumer); err != nil {
				log.Printf("Error en consumer: %v", err)
			}
			// Verificar si el contexto fue cancelado
			if ctx.Err() != nil {
				return
			}
		}
	}()

	log.Println("Consumidor (Group) Iniciado")

	// Esperar señal de terminación (Ctrl+C o Kubernetes Stop)
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	<-sigterm
	log.Println("Terminando consumidor")
	cancel()
	wg.Wait()
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

	// === 1. LÓGICA DE UN SOLO PRODUCTO (First-Seen) ===
	keyMonitoredName := fmt.Sprintf("producto_monitoreado_nombre:%s", nombreCat)
	
	// Intenta asignar el actual como el elegido
	seAsigno, _ := rdb.SetNX(ctx, keyMonitoredName, venta.ProductoID, 0).Result()
	if seAsigno {
		log.Printf("🔥 NUEVO ELEGIDO para %s: %s", nombreCat, venta.ProductoID)
	}

	// Consulta quién es el elegido oficial
	productoElegido, _ := rdb.Get(ctx, keyMonitoredName).Result()

	// Si coincide, guardamos en Stream
	if venta.ProductoID == productoElegido {
		keyStreamUnico := fmt.Sprintf("stream_precio_producto_unico:%s", nombreCat)
		rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: keyStreamUnico,
			MaxLen: 1000, 
			Values: map[string]interface{}{
				"precio": venta.Precio,
				"timestamp": time.Now().UnixMilli(),
			},
		})
	}

	// === 2. RESTO DE MÉTRICAS ===
	keyContador := fmt.Sprintf("contador:%s", nombreCat)
	keySumaCantidad := fmt.Sprintf("suma_cantidad:%s", nombreCat)
	keySumaPrecio := fmt.Sprintf("suma_precio:%s", nombreCat)
	
	nuevoContador, _ := rdb.Incr(ctx, keyContador).Result()
	nuevaSumaCant, _ := rdb.IncrBy(ctx, keySumaCantidad, int64(venta.CantidadVendida)).Result()
	nuevaSumaPrecio, _ := rdb.IncrByFloat(ctx, keySumaPrecio, venta.Precio).Result()

	if nuevoContador > 0 {
		promProd := float64(nuevaSumaCant) / float64(nuevoContador)
		rdb.Set(ctx, fmt.Sprintf("promedio_productos:%s", nombreCat), promProd, 0)

		promPrecio := nuevaSumaPrecio / float64(nuevoContador)
		rdb.Set(ctx, fmt.Sprintf("promedio_precio_tag:%s", nombreCat), promPrecio, 0)
	}

	rdb.Incr(ctx, "total_ventas")
	rdb.ZIncrBy(ctx, "ranking_productos", float64(venta.CantidadVendida), venta.ProductoID)
	rdb.ZIncrBy(ctx, fmt.Sprintf("ranking_productos_cat:%s", nombreCat), float64(venta.CantidadVendida), venta.ProductoID)

	currentMax, _ := rdb.Get(ctx, "precio_max_global").Float64()
	if venta.Precio > currentMax {
		rdb.Set(ctx, "precio_max_global", venta.Precio, 0)
	}
	
	currentMin, err := rdb.Get(ctx, "precio_min_global").Float64()
	if err == redis.Nil || venta.Precio < currentMin {
		rdb.Set(ctx, "precio_min_global", venta.Precio, 0)
	}
}