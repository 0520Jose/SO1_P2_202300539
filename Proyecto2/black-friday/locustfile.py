from locust import HttpUser, task, between
import random

class BlackFridayUser(HttpUser):
    wait_time = between(0.5, 2)

    @task(3)
    def comprar_belleza(self):
        productos = ["Labial-Matte", "Base-Liquida", "Perfume-Floral", "Crema-Hidratante"]
        self.client.post("/purchase", json={
            "categoria": 4,
            "producto_id": random.choice(productos),
            "precio": round(random.uniform(50.0, 300.0), 2),
            "cantidad_vendida": random.randint(1, 5)
        })

    @task(1)
    def comprar_electronica(self):
        self.client.post("/purchase", json={
            "categoria": 1,
            "producto_id": random.choice(["TV-Samsung", "iPhone-15", "Laptop-HP"]),
            "precio": round(random.uniform(2000.0, 8000.0), 2),
            "cantidad_vendida": 1
        })

    @task(1)
    def comprar_ropa(self):
        self.client.post("/purchase", json={
            "categoria": 2,
            "producto_id": random.choice(["Camisa-Polo", "Jeans-Levis", "Vestido-Gala"]),
            "precio": round(random.uniform(100.0, 500.0), 2),
            "cantidad_vendida": random.randint(1, 3)
        })

    @task(1)
    def comprar_hogar(self):
        self.client.post("/purchase", json={
            "categoria": 3,
            "producto_id": random.choice(["Licuadora", "Microondas", "Sofa"]),
            "precio": round(random.uniform(300.0, 1500.0), 2),
            "cantidad_vendida": 1
        })