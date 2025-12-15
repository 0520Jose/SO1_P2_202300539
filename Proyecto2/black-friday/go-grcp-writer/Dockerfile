# ===== Etapa de compilación =====
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Copiamos los archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el código fuente
COPY . .

# Compilamos el binario
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o producer main.go

# ===== Etapa final (imagen ligera) =====
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/producer .

EXPOSE 50051

CMD ["./producer"]
