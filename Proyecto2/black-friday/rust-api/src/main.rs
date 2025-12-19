use axum::{
    routing::post,
    Json, Router,
};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;

// Estructura de entrada (Desde Locust)
#[derive(Deserialize, Serialize, Debug)]
struct PurchaseInput {
    categoria: i32,
    producto_id: String,
    precio: f64,
    cantidad_vendida: i32,
}

// Estructura de respuesta
#[derive(Serialize, Deserialize)]
struct StatusResponse {
    status: String,
    exito: bool,
}

// Handler: Recibe de Locust -> Envía a Go-Bridge (HTTP)
async fn handle_purchase(
    Json(payload): Json<PurchaseInput>,
) -> Json<StatusResponse> {
    
    // URL del Bridge (nombre del servicio en K8s que definimos en el paso 1)
    let bridge_url = "http://go-bridge-service:80"; 
    let client = reqwest::Client::new();

    // Enviar POST al Bridge
    let res = client.post(format!("{}/forward", bridge_url))
        .json(&payload)
        .send()
        .await;

    match res {
        Ok(response) => {
            if response.status().is_success() {
                // Si el Bridge respondió OK, decodificamos su respuesta
                if let Ok(json_res) = response.json::<StatusResponse>().await {
                    Json(json_res)
                } else {
                    Json(StatusResponse {
                        status: "Error decodificando respuesta del Bridge".to_string(),
                        exito: false,
                    })
                }
            } else {
                Json(StatusResponse {
                    status: format!("Bridge respondió error: {}", response.status()),
                    exito: false,
                })
            }
        }
        Err(e) => {
            Json(StatusResponse {
                status: format!("No se pudo conectar al Bridge: {}", e),
                exito: false,
            })
        }
    }
}

#[tokio::main]
async fn main() {
    let app = Router::new()
        .route("/purchase", post(handle_purchase));

    let addr = SocketAddr::from(([0, 0, 0, 0], 3000));
    println!("Rust API (HTTP Client Mode) escuchando en {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}