use axum::{routing::post, Json, Router};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;

#[derive(Deserialize, Serialize, Debug)]
struct PurchaseInput {
    categoria: i32,
    producto_id: String,
    precio: f64,
    cantidad_vendida: i32,
}

#[derive(Serialize, Deserialize, Debug)]
struct BridgeResponse {
    estado: String,
}

#[derive(Serialize, Deserialize)]
struct StatusResponse {
    status: String,
}

async fn handle_purchase(Json(payload): Json<PurchaseInput>) -> Json<StatusResponse> {
    let bridge_url = "http://go-bridge-service:80";
    let client = reqwest::Client::new();

    let res = client
        .post(format!("{}/forward", bridge_url))
        .json(&payload)
        .send()
        .await;

    match res {
        Ok(response) => {
            if response.status().is_success() {
                match response.json::<BridgeResponse>().await {
                    Ok(bridge_res) => Json(StatusResponse {
                        status: bridge_res.estado,
                    }),
                    Err(_) => Json(StatusResponse {
                        status: "Decode Error".to_string(),
                    }),
                }
            } else {
                Json(StatusResponse {
                    status: "Bridge Error".to_string(),
                })
            }
        }
        Err(_) => Json(StatusResponse {
            status: "Connect Error".to_string(),
        }),
    }
}

#[tokio::main]
async fn main() {
    let app = Router::new().route("/purchase", post(handle_purchase));
    let addr = SocketAddr::from(([0, 0, 0, 0], 3000));
    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}