package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"      // <--- IMPORTANTE: Agregado para leer variables de entorno
	"strings" // <--- IMPORTANTE: Agregado para procesar la lista de brokers

	pb "grpc-writer/pb"

	"github.com/IBM/sarama"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedProductSaleServiceServer
	producer sarama.SyncProducer
}

func (s *server) ProcesarVenta(ctx context.Context, req *pb.ProductSaleRequest) (*pb.ProductSaleResponse, error) {
	msgBytes, err := json.Marshal(req)
	if err != nil {
		return &pb.ProductSaleResponse{Estado: "Error marshaling", Exito: false}, nil
	}

	msg := &sarama.ProducerMessage{
		Topic: "sales-topic",
		Value: sarama.StringEncoder(msgBytes),
	}

	_, _, err = s.producer.SendMessage(msg)
	if err != nil {
		return &pb.ProductSaleResponse{Estado: "Error Kafka", Exito: false}, nil
	}

	return &pb.ProductSaleResponse{Estado: "Procesado", Exito: true}, nil
}

func main() {
	// --- LÓGICA DE CONEXIÓN DINÁMICA ---
	// Leemos la variable de entorno que definimos en Kubernetes
	kafkaEnv := os.Getenv("KAFKA_BROKERS")
	if kafkaEnv == "" {
		kafkaEnv = "localhost:9092" // Valor por defecto para pruebas locales
	}
	// Convertimos el string "broker1:9092,broker2:9092" en una lista
	brokers := strings.Split(kafkaEnv, ",")
	
	log.Printf("🔌 Writer intentando conectar a Kafka en: %v", brokers)
	// -----------------------------------

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll

	// Aquí usamos la variable 'brokers' en lugar de "127.0.0.1:9092"
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalf("Failed to start Sarama producer: %v", err)
	}
	defer producer.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterProductSaleServiceServer(s, &server{producer: producer})

	log.Printf("🚀 Server gRPC listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}