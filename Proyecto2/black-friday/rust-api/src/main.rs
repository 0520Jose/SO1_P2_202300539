use axum::{
    routing::post,
    Json, Router,
};
use serde::Deserialize;
use std::net::SocketAddr;
use tonic::transport::Channel;

pub mod blackfriday {
    tonic::include_proto!("blackfriday");
}

use blackfriday::product_sale_service_client::ProductSaleServiceClient;
use blackfriday::{ProductSaleRequest, CategoriaProducto};

#[derive(Deserialize)]
struct PurchaseInput {
    categoria: i32,
    producto_id: String,
    precio: f64,
    cantidad_vendida: i32,
}

#[derive(serde::Serialize)]
struct StatusResponse {
    status: String,
    exito: bool,
}

async fn handle_purchase(
    axum::extract::State(mut grpc_client): axum::extract::State<ProductSaleServiceClient<Channel>>,
    Json(payload): Json<PurchaseInput>,
) -> Json<StatusResponse> {
    
    let request = tonic::Request::new(ProductSaleRequest {
        categoria: payload.categoria,
        producto_id: payload.producto_id,
        precio: payload.precio,
        cantidad_vendida: payload.cantidad_vendida,
    });

    match grpc_client.procesar_venta(request).await {
        Ok(response) => {
            let res = response.into_inner();
            Json(StatusResponse {
                status: res.estado,
                exito: res.exito,
            })
        }
        Err(e) => {
            Json(StatusResponse {
                status: format!("Error gRPC: {}", e),
                exito: false,
            })
        }
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let grpc_client = ProductSaleServiceClient::connect("http://localhost:50051").await?;

    let app = Router::new()
        .route("/purchase", post(handle_purchase))
        .with_state(grpc_client);

    let addr = SocketAddr::from(([0, 0, 0, 0], 3000));
    println!("Rust API listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}