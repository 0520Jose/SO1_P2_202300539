package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "go-bridge/pb"
)

type Venta struct {
	Categoria       int32   `json:"categoria"`
	ProductoID      string  `json:"producto_id"`
	Precio          float64 `json:"precio"`
	CantidadVendida int32   `json:"cantidad_vendida"`
}

func main() {
	grpcHost := os.Getenv("GRPC_HOST")
	if grpcHost == "" {
		grpcHost = "localhost:50051"
	}

	conn, err := grpc.NewClient(grpcHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	defer conn.Close()
	client := pb.NewProductSaleServiceClient(conn)

	r := gin.Default()
	r.POST("/forward", func(c *gin.Context) {
		var v Venta
		if err := c.ShouldBindJSON(&v); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		res, err := client.ProcesarVenta(ctx, &pb.ProductSaleRequest{
			Categoria:       pb.CategoriaProducto(v.Categoria),
			ProductoId:      v.ProductoID,
			Precio:          v.Precio,
			CantidadVendida: v.CantidadVendida,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"estado": res.Estado})
	})

	r.Run(":8080")
}