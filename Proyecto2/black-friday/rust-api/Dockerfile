FROM rust:1.82-slim-bookworm AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y protobuf-compiler libprotobuf-dev

RUN cargo new --bin rust-api
WORKDIR /app/rust-api

COPY Cargo.toml ./

COPY build.rs ./
COPY proto ./proto

RUN cargo build --release || true

COPY src ./src
RUN cargo build --release

FROM debian:bookworm-slim
WORKDIR /app

COPY --from=builder /app/rust-api/target/release/rust-api .

EXPOSE 3000
CMD ["./rust-api"]
