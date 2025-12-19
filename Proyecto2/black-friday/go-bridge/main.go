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

// Estructura igual a la que envía Rust
type Venta struct {
    Categoria       int32   `json:"categoria"`
    ProductoID      string  `json:"producto_id"`
    Precio          float64 `json:"precio"`
    CantidadVendida int32   `json:"cantidad_vendida"`
}

func main() {
    // Conexión gRPC hacia el Writer
    grpcHost := os.Getenv("GRPC_HOST")
    if grpcHost == "" {
        grpcHost = "localhost:50051"
    }

    conn, err := grpc.NewClient(grpcHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("No se pudo conectar al gRPC Server: %v", err)
    }
    defer conn.Close()
    client := pb.NewProductSaleServiceClient(conn)

    // Servidor HTTP (Gin) que escucha a Rust
    r := gin.Default()
    r.POST("/forward", func(c *gin.Context) {
        var v Venta
        if err := c.ShouldBindJSON(&v); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        // Convertir y Enviar por gRPC
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()

        res, err := client.ProcesarVenta(ctx, &pb.ProductSaleRequest{
            Categoria:       pb.CategoriaProducto(v.Categoria),
            ProductoId:      v.ProductoID,
            Precio:          v.Precio,
            CantidadVendida: v.CantidadVendida,
        })

        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Fallo gRPC", "detalles": err.Error()})
            return
        }

        c.JSON(http.StatusOK, gin.H{"estado": res.Estado, "exito": res.Exito})
    })

    r.Run(":8080") // Escuchar en puerto 8080
}