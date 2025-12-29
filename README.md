# SO1_P2_202300539

# Proyecto 2 - Arquitectura Distribuida en la Nube con Kubernetes

## 📋 RESUMEN DEL PROYECTO

Construir una **arquitectura distribuida en Kubernetes (GKE)** que simule el procesamiento de ventas de Black Friday en tiempo real.

---

## 🎯 OBJETIVO GENERAL

Construir una arquitectura de sistema distribuido genérico en Google Kubernetes Engine (GKE) para simular el procesamiento de información sobre ventas de black friday, aplicando conceptos de concurrencia, mensajería, almacenamiento en memoria y visualización, y comparando el rendimiento de diferentes tecnologías clave.

---

## 🏗️ ARQUITECTURA GENERAL

### Flujo de Datos
```
Locust (genera carga) 
  → Ingress NGINX 
  → API REST (Rust) 
  → Servicio Go (gRPC client) 
  → Servicios Go (gRPC servers) 
  → Kafka 
  → Consumidores Go 
  → Valkey (BD en memoria) 
  → Grafana (visualización)
```

---

## 🔧 COMPONENTES A IMPLEMENTAR

### 1. Locust (Generador de Carga)
- Genera tráfico HTTP hacia el Ingress
- Envía datos JSON con estructura:
  - `categoria` (Electronica, Ropa, Hogar, Belleza)
  - `producto_id`
  - `precio`
  - `cantidad_vendida`

### 2. Ingress NGINX
- Punto de entrada al clúster
- Enruta tráfico hacia la API REST

### 3. API REST en Rust
- **Función:** Recibe peticiones HTTP de Locust
- **Acción:** Envía datos a Deployment Go #1
- **Escalabilidad:** HPA configurado
  - Rango: 1-3 réplicas
  - Trigger: CPU > 30%

### 4. Servicios en Go (3 Deployments)

#### Deployment 1: API REST + gRPC Client
- Recibe datos del API Rust
- Actúa como cliente gRPC
- Invoca funciones para publicar en Kafka

#### Deployment 2: gRPC Server (Writer 1)
- Implementa servicio gRPC
- Publica mensajes en Kafka
- **Configuración:** Probar con 1 réplica

#### Deployment 3: gRPC Server (Writer 2)
- Implementa servicio gRPC
- Publica mensajes en Kafka
- **Configuración:** Probar con 2 réplicas

### 5. Kafka (Message Broker)
- **Implementación:** Strimzi Kafka
- Almacena y distribuye mensajes
- Comunicación asíncrona entre servicios

### 6. Consumidor Go
- Consume mensajes de Kafka
- Procesa datos
- Almacena información en Valkey

### 7. Valkey (Base de Datos en Memoria)
- **Configuración:** 2 réplicas por defecto
- Persistencia de datos asegurada
- **Implementación:** Usar KubeVirt
- Almacena datos procesados

### 8. Grafana
- Visualiza datos de Valkey
- **Instalación:** Usar Helm
- Dashboard con métricas del sistema

### 9. Zot (Container Registry)
- **Ubicación:** VM en GCP (fuera del clúster K8s)
- Registry privado para imágenes Docker
- Todas las imágenes se publican y descargan desde Zot
- Soporte para OCI Artifacts

---

## 📝 ESTRUCTURA gRPC (Proto)

```protobuf
syntax = "proto3";
package blackfriday;
option go_package = "./proto";

// Mensaje con información de venta de producto durante Black Friday
message ProductSaleRequest {
  CategoriaProducto categoria = 1;
  string producto_id = 2;
  double precio = 3;
  int32 cantidad_vendida = 4;
}

// Lista de categorías de productos
enum CategoriaProducto {
  Electronica = 1;
  Ropa = 2;
  Hogar = 3;
  Belleza = 4;
}

// Respuesta del servidor
message ProductSaleResponse {
  string estado = 1;
}

// Servicio gRPC para procesamiento de ventas durante Black Friday
service ProductSaleService {
  rpc ProcesarVenta (ProductSaleRequest) returns (ProductSaleResponse);
}
```

---

## 📊 DASHBOARD DE GRAFANA

### Asignación de Categoría por Carnet
Según el **último dígito de tu carnet**:
- **0, 1, 2** → Electronica
- **3, 4, 5** → Ropa
- **6, 7** → Hogar
- **8, 9** → Belleza

### Gráfica Requerida
**Tipo:** Gráfica de Barras

**Contenido:** Total de reportes por categoría (número de veces que se registró cada categoría)

---

## 💻 TECNOLOGÍAS OBLIGATORIAS

| Categoría | Tecnología |
|-----------|------------|
| ☁️ Cloud | Google Cloud Platform (GCP) |
| 🎛️ Orquestación | Google Kubernetes Engine (GKE) |
| 🐳 Contenedores | Docker |
| 🦀 API REST | Rust |
| 🐹 Servicios | Go |
| 🐍 Generación de Carga | Locust (Python) |
| 📨 Message Broker | Kafka (Strimzi) |
| 💾 BD en Memoria | Valkey |
| 📊 Visualización | Grafana |
| 🎯 Ingress | NGINX Ingress Controller |
| 📦 Container Registry | Zot |
| 🖥️ Gestión de VMs | KubeVirt |

---

## 📦 ENTREGABLES

### 1. Repositorio GitHub (Privado)

**Estructura:**
```
proyecto2/
├── rust-api/
│   ├── src/
│   ├── Dockerfile
│   └── Cargo.toml
├── go-services/
│   ├── grpc-client/
│   ├── grpc-server-1/
│   ├── grpc-server-2/
│   ├── consumer/
│   └── proto/
├── kubernetes/
│   ├── deployments/
│   ├── services/
│   ├── ingress/
│   ├── hpa/
│   └── kafka/
├── locust/
│   └── locustfile.py
├── scripts/
└── README.md
```

**Contenido:**
- Código fuente (Rust, Go, Python)
- YAMLs de Kubernetes
  - Deployments
  - Services
  - Ingress
  - HPA
  - Configuraciones de Kafka
- Dockerfiles para cada componente
- Scripts de apoyo
- **Importante:** Agregar al auxiliar como colaborador

### 2. Informe Técnico (Markdown)

**Debe incluir:**

1. **Documentación de Deployments**
   - Descripción de cada componente
   - Ejemplos de configuración
   - Comandos de despliegue

2. **Instrucciones de Despliegue**
   - Requisitos previos
   - Pasos claros para desplegar todo el sistema
   - Comandos de verificación

3. **Arquitectura del Sistema**
   - Diagrama del flujo de datos
   - Explicación de cada componente
   - Interacciones entre servicios

4. **Comparativas de Rendimiento**
   - Kafka bajo diferentes cargas
   - Valkey con diferentes números de réplicas
   - API REST (Rust) vs gRPC (Go)
   - Métricas y gráficas

5. **Proceso de Desarrollo**
   - Metodología utilizada
   - Retos encontrados
   - Soluciones implementadas

6. **Conclusiones**
   - Aprendizajes clave
   - Recomendaciones
   - Posibles mejoras

---

## ⚙️ CONFIGURACIONES CLAVE

### HPA (Horizontal Pod Autoscaler)
```yaml
# API Rust
replicas: 1-3
trigger: CPU > 30%
```

### Réplicas de Valkey
```yaml
replicas: 2
persistencia: habilitada
```

### Pruebas de gRPC Servers
- **Server 1:** 1 réplica
- **Server 2:** 2 réplicas
- Comparar rendimiento

### Namespaces
- Organizar componentes en namespaces
- Separación lógica de recursos

### OCI Artifact
- Descargar archivo de entrada desde Zot
- Documentar qué archivo y cómo se usa

---

## 🎯 OBJETIVOS DE APRENDIZAJE

Al completar este proyecto serás competente en:

1. ✅ Diseñar arquitecturas de microservicios en la nube
2. ✅ Orquestar contenedores con Kubernetes
3. ✅ Desarrollar servicios concurrentes en Go y Rust
4. ✅ Configurar message brokers (Kafka)
5. ✅ Integrar bases de datos en memoria (Valkey)
6. ✅ Implementar Container Registry privado
7. ✅ Analizar y comparar rendimiento de componentes
8. ✅ Generar carga de pruebas con Locust
9. ✅ Visualizar métricas con Grafana

---

## 📋 CHECKLIST DE IMPLEMENTACIÓN

### Infraestructura
- [ ] Crear clúster GKE en GCP
- [ ] Configurar VM para Zot
- [ ] Instalar NGINX Ingress Controller
- [ ] Configurar namespaces

### Desarrollo
- [ ] API REST en Rust
- [ ] Deployment Go #1 (gRPC Client)
- [ ] Deployment Go #2 (gRPC Server - 1 réplica)
- [ ] Deployment Go #3 (gRPC Server - 2 réplicas)
- [ ] Consumidor Go
- [ ] Locust (generador de carga)

### Message Broker & Storage
- [ ] Desplegar Kafka (Strimzi)
- [ ] Desplegar Valkey (2 réplicas)
- [ ] Configurar persistencia

### Visualización
- [ ] Instalar Grafana con Helm
- [ ] Crear dashboard
- [ ] Configurar gráfica de barras por categoría

### Docker & Registry
- [ ] Crear Dockerfiles
- [ ] Construir imágenes
- [ ] Publicar en Zot
- [ ] Configurar pull desde Zot

### Kubernetes
- [ ] YAMLs de Deployments
- [ ] YAMLs de Services
- [ ] YAML de Ingress
- [ ] YAML de HPA (API Rust)
- [ ] Configuraciones de Kafka

### Pruebas
- [ ] Pruebas de carga con Locust
- [ ] Verificar escalado automático
- [ ] Comparar rendimiento Kafka
- [ ] Comparar rendimiento Valkey
- [ ] Comparar REST vs gRPC

### Documentación
- [ ] README con instrucciones
- [ ] Informe técnico completo
- [ ] Diagramas de arquitectura
- [ ] Conclusiones y comparativas

---

## 🚀 RECOMENDACIONES OPCIONALES

Aunque no son puntuables, se recomienda:

1. **Optimización de Recursos**
   - Configurar `requests` y `limits` para cada pod
   - Evitar saturación del sistema

2. **Gestión de Datos**
   - Implementar tiempo de expiración (TTL) en Valkey
   - Evitar crecimiento descontrolado de datos

3. **Monitoreo**
   - Agregar métricas adicionales en Grafana
   - Configurar alertas

4. **Seguridad**
   - Usar secrets para credenciales
   - Configurar RBAC en Kubernetes

---

## ⚠️ RESTRICCIONES

- ❌ Proyecto **INDIVIDUAL**
- ✅ Uso **OBLIGATORIO** de Locust
- ✅ Uso **OBLIGATORIO** de GKE
- ✅ Repositorio GitHub **PRIVADO**
- ✅ Agregar al auxiliar como colaborador

---

## 📚 COMPETENCIAS DESARROLLADAS

Este proyecto te permitirá desarrollar competencias en:

- Arquitectura de sistemas distribuidos
- Cloud Computing (GCP)
- Orquestación de contenedores (Kubernetes)
- Programación concurrente (Go, Rust)
- Message-driven architecture
- Persistencia en memoria
- Monitoreo y visualización
- DevOps y CI/CD
- Análisis de rendimiento

---

## 🎓 CURSO

**Sistemas Operativos 1**
- Universidad San Carlos de Guatemala
- Facultad de Ingeniería
- Ingeniería en Ciencias y Sistemas


 Toma en cuenta toda la estructura de conexiones necesarias, el usuario invoca a locus localmente, locus debe hacer el push trafic hacia el ingress este lo envia al deployment api rest en rost, luego la api rest encia hacia el otro deployment que es una api rest grpc client que tien de 1 a 3 replicas esta en go luego esta envia havia otro grpc server que se un deployment este contiene el kafka writer esta en go este publica hacia otro deployment que es el kafka Strrimzi y esta es consumida por el kafka consumer que es un deploymente en go con 1 a 2 replicas este envia hacia otro deploymente que tien como entrada kubevirt es valkey db y esta es consumida por el deployment grafana, neceisto que todo este flujo se cumpla ademas debes tomar en cuenta que por lo que entinedo el usuario debe subir las imagenes hacia una maquina virutal con un zot container registry y desde ahi la nube debe realizar el pull container images, todo esto se debe cumplir a cabalidad, no puedo cambiar, omitar nada, necesito que revices que todo se cumpla y si hay algo que falte genera una ruta para realizarlo paso a paso todo dentro de aws