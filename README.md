# MiniShop: High-Performance Flash Sale System

MiniShop is a high-performance flash sale system built on a modern **microservices architecture**, designed to handle large-scale concurrent traffic. It incorporates flow control, graceful degradation, message queues, and distributed locking to effectively handle the challenges of flash sale scenarios.


## 🏗️ Tech Stack

### Backend Technologies
- **Go 1.21** – Main programming language, known for high performance in concurrent environments
- **Gin** – Lightweight HTTP framework for building RESTful APIs
- **GORM** – ORM library that simplifies database operations
- **Redis** – In-memory cache and distributed lock management; supports atomic Lua scripts
- **PostgreSQL** – Relational database providing strong ACID guarantees
- **RabbitMQ / Kafka** – Message queue systems for asynchronous traffic smoothing
- **Docker** – Containerization for consistent deployment
- **Nginx** – Reverse proxy and load balancer

### Frontend Technologies
- **React 18** – Modern front-end framework
- **TypeScript** – Strongly-typed JavaScript for safer development
- **Tailwind CSS** – Utility-first CSS framework
- **Zustand** – Lightweight state management library
- **TanStack Query** – Data fetching and caching solution
- **Axios** – Promise-based HTTP client

### Monitoring & Deployment
- **Prometheus + Grafana** – Real-time monitoring and alerting
- **Docker Compose** – Local development environment orchestration
- **Kubernetes (optional)** – Production-level container orchestration support


## 🚀 Core Features

### Flash Sale Capabilities
- **Atomic Stock Deduction** – Redis Lua scripts ensure atomic inventory changes
- **User Deduplication** – Prevents duplicate purchases and overselling
- **Asynchronous Order Processing** – Order creation is handled via message queues for responsiveness
- **Multi-level Rate Limiting** – Token bucket, sliding window, and other strategies
- **Circuit Breaker & Fallback** – Automatic failure detection and service degradation
- **Distributed Locking** – Redis-based locking to avoid concurrent conflicts

### Advanced Features
- **Eventual Consistency** – Ensures Redis cache and PostgreSQL remain consistent
- **Idempotent Design** – Prevents duplicated operations under retry conditions
- **Real-time Monitoring** – Performance and health metrics available through Grafana dashboards
- **Graceful Shutdown** – Smooth restarts with resource cleanup
- **Distributed Tracing** – End-to-end request tracking across services


## 📁 System Architecture



### Microservices Overview
- **API Gateway** – Unified entry point for authentication, rate limiting, and routing
- **Seckill Service** – Core flash sale logic with high-concurrency handling
- **Order Service** – Asynchronous creation and management of orders
- **Inventory Service** – Manages stock consistency and synchronization
- **Cache Service** – Redis abstraction layer for locking and cache operations


## 🚀 Getting Started

### Prerequisites
- **Go** >= 1.21
- **Node.js** >= 16.0
- **Docker** >= 20.0
- **Docker Compose** >= 2.0


## 🎯 Feature Breakdown

### 1. High-Concurrency Handling
- **Atomic Stock Deduction** – Redis Lua script prevents overselling
- **Token Bucket Algorithm** – Smooth rate limiting for burst traffic
- **Circuit Breaker Pattern** – Automatic fallback and degradation
- **Asynchronous Queueing** – Message queues buffer and smooth spikes

### 2. Data Consistency
- **Eventual Consistency** – Cache and DB eventually synchronized
- **Distributed Transactions** – Cross-service coordination where needed
- **Idempotency** – Prevents duplicated operations on retries
- **Compensation Mechanism** – Fallback logic for retries or manual intervention

### 3. Monitoring & Observability
- **Metrics Collection** – Prometheus scrapes key system indicators
- **Dashboard Visualization** – Grafana displays real-time system health
- **Health Checks** – Liveness/readiness probes for each service
- **Distributed Tracing** – Track requests across service boundaries

### 4. User Experience
- **Countdown Timer** – Millisecond-level accurate countdown for events
- **Status Feedback** – Clear success/failure indicators
- **Responsive UI** – Optimized for desktop and mobile
- **Offline Support** – PWA capabilities for offline access


## ⚙️ Performance Benchmarks

- **QPS**: > 10,000 requests/second
- **Concurrent Users**: > 50,000
- **Response Time**: < 100ms (P99)
- **Availability**: > 99.9%


## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.


## 🙏 Acknowledgements

Thanks to all contributors and the open-source community who helped make this project possible.


👉 If you found this project helpful, please ⭐ it and share it with others!

