FROM golang:1.24.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CAMBIO AQUÍ: Nombre del binario 'bridge'
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bridge main.go

FROM alpine:latest

WORKDIR /root/

# CAMBIO AQUÍ: Copiamos 'bridge'
COPY --from=builder /app/bridge .

EXPOSE 8080

CMD ["./bridge"]