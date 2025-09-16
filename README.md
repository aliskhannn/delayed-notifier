# DelayedNotifier

**DelayedNotifier** is a backend service for scheduling and sending delayed notifications via queues (RabbitMQ).  
It allows you to create notifications that should be delivered at a specific time via multiple channels (Email, Telegram).

---

## Features

- **HTTP API** for creating, cancelling, and checking notifications
- **Background workers** consume messages from RabbitMQ and send notifications at the right time
- **Retry mechanism** with exponential backoff in case of delivery failures
- **Channels supported:** Email, Telegram
- **Redis caching** for fast status checks
- **Simple frontend** (port **3000**) to test the service via a UI

---

## Project Structure

```bash
.
├── backend/                 # Backend service
│   ├── cmd/                 # Application entry points
│   ├── config/              # Configuration files
│   ├── internal/            # Internal application packages
│   │   ├── api/             # HTTP handlers, router, server
│   │   ├── config/          # Config parsing logic
│   │   ├── middlewares/     # HTTP middlewares
│   │   ├── mocks/           # Generated mocks for testing
│   │   ├── model/           # Data models
│   │   ├── rabbitmq/        # RabbitMQ connection and consumers
│   │   ├── repository/      # Database repositories
│   │   ├── service/         # Business logic
│   │   └── worker/          # Background workers for scheduled delivery
│   ├── migrations/          # Database migrations
│   ├── pkg/                 # External clients (Email, Telegram)
│   ├── Dockerfile           # Backend Dockerfile
│   ├── rabbitmq.dockerfile  # RabbitMQ Dockerfile with plugins
│   ├── go.mod
│   └── go.sum
├── frontend/                # Frontend application
├── plugins/                 # RabbitMQ plugins
├── .env.example             # Example environment variables
├── docker-compose.yml       # Multi-service Docker setup
├── Makefile                 # Development commands
└── README.md
````

---

## Makefile Commands

```make
# Run all backend tests with verbose output
make test

# Run linters (vet + golangci-lint)
make lint

# Build and start all Docker services
make docker-up

# Stop and remove all Docker services and volumes
make docker-down
```

---

## Configuration (`.env`)

Before running the project, copy `.env.example` to `.env` and set your own values:

```bash
cp .env.example .env
```

#### 🔑 Notes:

* **SMTP credentials**: Create an account, for example, on [Mailtrap](https://mailtrap.io/) and copy the SMTP login + password into `.env`.
* **Telegram Chat ID**: Open Telegram, start your bot, then go to `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates` and find your `chat.id`.

---

## Running the Project

1. Copy and update `.env`:

   ```bash
   cp .env.example .env
   ```

2. Build and run services via Docker:

   ```bash
   make docker-up
   ```

3. The backend will be available at:

    * **Backend API** → `http://localhost:8080/api/notify`
    * **Frontend UI** → `http://localhost:3000`

4. To stop services:

   ```bash
   make docker-down
   ```

---

## API Endpoints

All endpoints are available under `/api/notify`:

| Method | Endpoint | Description                  |
| ------ | -------- | ---------------------------- |
| POST   | `/`      | Create a new notification    |
| GET    | `/`      | Get all notifications        |
| GET    | `/:id`   | Get status of a notification |
| DELETE | `/:id`   | Cancel a notification        |

---

## Example Requests

### 1. Create a Notification

**POST** `http://localhost:8080/api/notify/`

Request body:

```json
{
  "message": "Reminder: Standup meeting at 10:00",
  "send_at": "2025-09-16 10:00:00",
  "retries": 3,
  "to": "123456789",
  "channel": "telegram"
}
```

Response:

```json
{
  "id": "c3fcd3d7-4a8f-43d5-a289-6f3d0b2f9f5b"
}
```

---

### 2. Get Notification Status

**GET** `http://localhost:8080/api/notify/c3fcd3d7-4a8f-43d5-a289-6f3d0b2f9f5b`

Response:

```json
{
  "status": "pending"
}
```

---

### 3. Get All Notifications

**GET** `http://localhost:8080/api/notify/`

Response:

```json
[
  {
    "id": "c3fcd3d7-4a8f-43d5-a289-6f3d0b2f9f5b",
    "message": "Reminder: Standup meeting at 10:00",
    "send_at": "2025-09-16T10:00:00Z",
    "status": "pending",
    "to": "123456789",
    "channel": "telegram"
  }
]
```

---

### 4. Cancel a Notification

**DELETE** `http://localhost:8080/api/notify/c3fcd3d7-4a8f-43d5-a289-6f3d0b2f9f5b`

Response:

```json
{
  "message": "notification cancelled"
}
```

---

## Frontend

A simple UI is available at **[http://localhost:3000](http://localhost:3000)**.
It provides:

* A form to create a notification
* A table with all notifications and their statuses
* Buttons to cancel a notification

---

## Summary

* **Backend** (Go + RabbitMQ + PostgreSQL + Redis) → runs on **port 8080**
* **Frontend** → runs on **port 3000**
* Notifications can be created via **API or UI**
* Notifications are delivered via **Email (SMTP)** and **Telegram Bot**
* Failed deliveries are retried automatically

```