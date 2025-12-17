package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"

	pb "grpc-writer/pb"

	"github.com/IBM/sarama"
	"google.golang.org/grpc"
)

// Estructura del servidor gRPC
type server struct {
	pb.UnimplementedProductSaleServiceServer
	producer sarama.SyncProducer
}

// Implementación del método ProcesarVenta
func (s *server) ProcesarVenta(ctx context.Context, req *pb.ProductSaleRequest) (*pb.ProductSaleResponse, error) {
	msgBytes, err := json.Marshal(req)
	if err != nil {
		return &pb.ProductSaleResponse{Estado: "Error marshaling", Exito: false}, nil
	}

	// Enviar el mensaje a Kafka
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

// Función principal para iniciar el servidor gRPC
func main() {
	kafkaEnv := os.Getenv("KAFKA_BROKERS")
	if kafkaEnv == "" {
		kafkaEnv = "localhost:9092"
	}
	brokers := strings.Split(kafkaEnv, ",")

	// Configuración del productor Sarama
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalf("Error al crear productor Sarama: %v", err)
	}
	defer producer.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Error al conectar: %v", err)
	}

	// Crear y registrar el servidor gRPC
	s := grpc.NewServer()
	pb.RegisterProductSaleServiceServer(s, &server{producer: producer})

	log.Printf("Server gRPC escuchando en: %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Error al conectar al servidor gRPC: %v", err)
	}
}