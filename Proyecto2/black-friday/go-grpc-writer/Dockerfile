FROM golang:1.24.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o producer main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/producer .

EXPOSE 50051

CMD ["./producer"]
