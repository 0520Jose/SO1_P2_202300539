package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"

	pb "go-grpc-writer/pb"

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
		return &pb.ProductSaleResponse{Estado: "Error marshaling"}, nil
	}

	msg := &sarama.ProducerMessage{
		Topic: "sales-topic",
		Value: sarama.StringEncoder(msgBytes),
	}

	_, _, err = s.producer.SendMessage(msg)
	if err != nil {
		return &pb.ProductSaleResponse{Estado: "Error Kafka"}, nil
	}

	return &pb.ProductSaleResponse{Estado: "Procesado"}, nil
}

func main() {
	kafkaEnv := os.Getenv("KAFKA_BROKERS")
	if kafkaEnv == "" {
		kafkaEnv = "localhost:9092"
	}
	brokers := strings.Split(kafkaEnv, ",")

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalf("Fatal Kafka: %v", err)
	}
	defer producer.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Fatal Listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterProductSaleServiceServer(s, &server{producer: producer})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fatal Serve: %v", err)
	}
}