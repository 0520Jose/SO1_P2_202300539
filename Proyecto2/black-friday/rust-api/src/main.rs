// Rust API que recibe solicitudes HTTP y se comunica con el servicio gRPC

// Dependencias de Axum y Tonic 
use axum::{
    routing::post,
    Json, Router,
};

// Dependencias para serialización y deserialización
use serde::Deserialize;
use std::net::SocketAddr;
use tonic::transport::Channel;


// Importar el cliente gRPC generado por Tonic
pub mod blackfriday {
    tonic::include_proto!("blackfriday");
}

// Usar el cliente gRPC
use blackfriday::product_sale_service_client::ProductSaleServiceClient;
use blackfriday::{ProductSaleRequest, CategoriaProducto};

// Estructuras para manejar la entrada JSON
#[derive(Deserialize)]
struct PurchaseInput {
    categoria: i32,
    producto_id: String,
    precio: f64,
    cantidad_vendida: i32,
}

// Estructura para la respuesta JSON
#[derive(serde::Serialize)]
struct StatusResponse {
    status: String,
    exito: bool,
}

async fn handle_purchase(
    axum::extract::State(mut grpc_client): axum::extract::State<ProductSaleServiceClient<Channel>>,
    Json(payload): Json<PurchaseInput>,
) -> Json<StatusResponse> {
    
    // Crear la solicitud gRPC
    let request = tonic::Request::new(ProductSaleRequest {
        categoria: payload.categoria,
        producto_id: payload.producto_id,
        precio: payload.precio,
        cantidad_vendida: payload.cantidad_vendida,
    });

    // Llamar al método gRPC y manejar la respuesta
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

// Función principal para iniciar el servidor Axum
#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Configurar el cliente gRPC
    let grpc_host = std::env::var("GRPC_HOST")
        .unwrap_or_else(|_| "http://localhost:50051".to_string());
    let grpc_client = ProductSaleServiceClient::connect(grpc_host).await?;

    // Configurar la aplicación Axum
    let app = Router::new()
        .route("/purchase", post(handle_purchase))
        .with_state(grpc_client);

    let addr = SocketAddr::from(([0, 0, 0, 0], 3000));
    println!("Rust API escuchando en {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}