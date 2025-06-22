# RADIUS Accounting Server

This project implements a RADIUS Accounting Server and a Redis Subscriber Logger. The RADIUS server listens for `Accounting-Request` packets, stores the accounting data in Redis, and a separate service subscribes to Redis key updates and logs each accounting event to persistent log files.

---

## About RADIUS

RADIUS (Remote Authentication Dial-In User Service) is a protocol commonly used for authentication, authorization, and accounting (AAA). This project focuses on the **Accounting** part, where a Network Access Server (NAS) sends accounting packets to a RADIUS server.

Each accounting packet includes metadata such as:
- User identity
- Session ID
- Start, Stop, or Interim-Update status
- NAS details
- Client IP address

The server receives these packets over UDP, parses them, and stores the extracted data in Redis. A separate logging service watches Redis for updates and logs structured accounting events.

---

## Project Structure

- `cmd/` — Entrypoints for individual services:
  - `radius-controlplane` — the RADIUS accounting server
  - `redis-controlplane-logger` — the Redis key event subscriber and logger

- `internal/` — Reusable packages:
  - `config/` — Environment-based configuration
  - `logger/` — Logrus-based logging abstraction
  - `model/` — Shared struct definitions
  - `redisclient/` — Redis storage implementation
  - `server/` — RADIUS UDP server

- `docker/` — Contains:
  - `docker-compose.yml` — Compose configuration
  - `Dockerfile.*` — Dockerfile per service
  - `persisted_logs/` — Volume-mounted log storage

- `testdata/` — Input files used for testing (e.g., `test_acct_start.txt`)

---

# Getting Started

## Infrastructure Diagram

```
                      +----------------+
                      | radclient-test |
                      +--------+-------+
                               |
                               v
                    +----------+------------+
                    |  radius-controlplane  |
                    +----------+------------+
                               |
                               v
                         [ Redis Server ]
                               |
                               v
                +-------------------------------+
                | redis-controlplane-logger     |
                | - Subscribes to Redis keyspace|
                | - Logs updates to file        |
                +-------------------------------+
```

---

## Prerequisites

1. Docker Engine and Docker Compose

   On macOS, you can use [Colima](https://smallsharpsoftwaretools.com/tutorials/use-colima-to-run-docker-containers-on-macos/):
   ```bash
   colima start
   ```

2. GNU Make
   ```bash
   sudo apt-get install build-essential
   ```

---

## Configuration

Configuration for the RADIUS server and Redis subscriber logger is handled via environment variables defined in `docker/docker-compose.yml`.

### `radius-controlplane` service

```yaml
environment:
  - RADIUS_SECRET=testing123
  - REDIS_ADDR=redis:6379
  - REDIS_PASSWORD=
  - REDIS_DB=0
  - SERVER_PORT=:1813
  - LOG_FILE_PATH=/var/log/radius_server.log
```

| Variable         | Description                                                         |
|------------------|---------------------------------------------------------------------|
| `RADIUS_SECRET`  | Shared secret used to parse and authenticate incoming RADIUS packets |
| `REDIS_ADDR`     | Redis server address used for storing accounting records            |
| `REDIS_PASSWORD` | Password for Redis (leave empty if not used)                        |
| `REDIS_DB`       | Redis database index (usually 0)                                    |
| `SERVER_PORT`    | UDP port the server listens on, must include colon (e.g., `:1813`)  |
| `LOG_FILE_PATH`  | Log file path inside container for writing logs                     |

---

### `redis-controlplane-logger` service

```yaml
environment:
  - REDIS_ADDR=redis:6379
  - REDIS_PASSWORD=
  - REDIS_DB=0
  - LOG_FILE_PATH=/var/log/redis_logger.log
```

| Variable         | Description                                                       |
|------------------|-------------------------------------------------------------------|
| `REDIS_ADDR`     | Redis address to subscribe to                                     |
| `REDIS_PASSWORD` | Redis password (if any)                                           |
| `REDIS_DB`       | Redis DB index                                                    |
| `LOG_FILE_PATH`  | File where Redis logger writes accounting update logs             |

---

You can also override these variables using a `.env` file in the root of the project. These are automatically loaded via `github.com/joho/godotenv/autoload`.


## Set Up

1. Clone the repository and move to the root project directory:

   ```bash
   git clone https://github.com/mohit83k/radius.git
   cd radius
   ```

2. Start the infrastructure:

   ```bash
   make up
   ```

   This builds and starts:
   - RADIUS server
   - Redis
   - Logger
   - radclient-test container

3. Access the `radclient-test` container:

   ```bash
   make radclient-bash
   ```

4. Inside the container, run a test RADIUS request:

   ```bash
   radclient -x radius-controlplane:1813 acct testing123 < test_acct_start.txt
   ```

---

## Integration Test

To run the full automated integration test:

```bash
make integration-test
```

This will:

- Rotate and reset log files
- Restart services
- Run a concurrent test script from `radclient-test` container
- Check log correctness (expecting 9 accounting entries for 3 users × 3 request types)

---

## Other Make Commands

| Command | Description |
|--------|-------------|
| `make build` | Build Go binaries locally |
| `make up` | Build and start Docker Compose environment |
| `make down` | Stop and remove containers |
| `make restart` | Restart all containers |
| `make test` | Run all unit tests |
| `make tidy` | Run `go mod tidy` |
| `make clean` | Clean Go binaries and cache |
| `make clean-logs` | Delete all logs from `docker/persisted_logs/` |
| `make rotate-logs` | Rotate `radius_server.log` and `redis_logger.log` with timestamp |
| `make logs` | Follow logs from all containers |
| `make logs-controlplane` | Follow logs from RADIUS server only |
| `make radclient-bash` | Open a bash shell in the radclient-test container |
| `make run-test-script` | Run the test script inside radclient-test container |
| `make check-test-script-logs` | Verify logs contain exactly 9 accounting entries |
| `make restart-logger` | Restart only the Redis logger service |
| `make ps` | Show running container status |
| `make integration-test-build` | Clean, rebuild, and run integration test |
| `make integration-test-up` | Start infra and run test (no rebuild) |