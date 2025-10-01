# BDX Exporter

A Prometheus exporter service written in Golang that collects monitoring metrics from BDX (360°View Data Center Information Management) dashboards and exposes them in Prometheus-compatible format.

## Project Overview

The BDX Exporter is designed to scrape data from HTTP endpoints and web pages to provide comprehensive monitoring metrics for data center infrastructure. It collects temperature and humidity data, CDU (Cooling Distribution Unit) status information, and liquid cooling metrics from the 360View monitoring system.

The exporter transforms raw data from various dashboard endpoints into standardized Prometheus metrics, enabling monitoring and alerting for data center environmental conditions and cooling system performance.

## Features and Capabilities

- **Temperature & Humidity Monitoring**: Collects real-time temperature and humidity readings from multiple sensors
- **CDU Status Monitoring**: Scrapes alarm and parameter data from CDU dashboards
- **Liquid Cooling Monitoring**: Extracts metrics from liquid cooling overview dashboards including CDU statuses and rack energy valve information
- **Prometheus Integration**: Exposes metrics in standard Prometheus format
- **Health Checks**: Provides health status endpoint for monitoring exporter health
- **Configurable Scraping**: Adjustable scrape intervals and timeouts
- **Docker Support**: Containerized deployment with minimal resource requirements
- **Graceful Shutdown**: Proper signal handling for clean shutdowns

## Installation

### Prerequisites

- Go 1.19+ (for building from source)
- Docker (for containerized deployment)

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/reski-rukmantiyo/bdx-prometheus.git
cd bdx-prometheus

# Install dependencies
go mod download

# Build the application
go build -o bdx-exporter .

# Run the exporter
./bdx-exporter
```

### Option 2: Docker Deployment

```bash
# Build the Docker image
docker build -t bdx-exporter .

# Run the container
docker run -p 8080:8080 bdx-exporter
```

## Configuration

The exporter is configured via environment variables. Create a `.env` file in the project root or set environment variables directly.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Port on which the exporter listens |
| `SCRAPE_INTERVAL` | `30s` | Interval between metric collections |
| `HTTP_TIMEOUT` | `10s` | Timeout for HTTP requests |
| `SCRAPE_TIMEOUT` | `30s` | Timeout for scraping operations |
| `TRH_URL` | `https://app.managed360view.com/360view/trh_monitoring_dashboard.php` | URL for temperature and humidity data |
| `LIQUID_URL` | `https://app.managed360view.com/360view/liquid_cooling_overview.php` | URL for liquid cooling overview |
| `CDU_URLS` | Comma-separated list of CDU dashboard URLs | URLs for individual CDU dashboards |
| `SESS_MAP` | Default session map | Session cookie value for authentication |
| `PHPSESSID` | Default PHP session ID | PHP session cookie value for authentication |
| `REFERER` | `https://app.managed360view.com/360view/trh_monitoring_dashboard.php` | Referer header for requests |

### Example .env File

```env
PORT=8080
SCRAPE_INTERVAL=30s
HTTP_TIMEOUT=10s
SCRAPE_TIMEOUT=30s
TRH_URL=https://app.managed360view.com/360view/trh_monitoring_dashboard.php
LIQUID_URL=https://app.managed360view.com/360view/liquid_cooling_overview.php
CDU_URLS=https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38329,https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38337
SESS_MAP=your_session_map_here
PHPSESSID=your_php_session_id_here
REFERER=https://app.managed360view.com/360view/trh_monitoring_dashboard.php
```

### Authentication

The exporter requires valid session cookies to access the BDX dashboards. These must be obtained from a valid login session to the 360View application.

## Usage Examples

### Basic Usage

```bash
# Start the exporter
./bdx-exporter

# Or with Docker
docker run -p 8080:8080 -e SESS_MAP=your_session -e PHPSESSID=your_session_id bdx-exporter
```

### Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'bdx-exporter'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 30s
```

### Health Check

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "last_collect": "2025-10-01T12:00:00Z",
  "last_success": true
}
```

## API Endpoints Documentation

### Health Check Endpoint

**GET /health**

Returns the health status of the exporter.

**Response:**
```json
{
  "status": "healthy|unhealthy",
  "last_collect": "RFC3339 timestamp",
  "last_success": true|false
}
```

### Metrics Endpoint

**GET /metrics**

Exposes Prometheus metrics in the standard format.

## Prometheus Metrics Documentation

### Temperature & Humidity Metrics

#### `bdx_temperature`
- **Type**: Gauge
- **Description**: Current temperature reading in Celsius
- **Labels**:
  - `name`: Sensor identifier
- **Example**:
  ```
  bdx_temperature{name="CGK3A-EMS-1.04-TH-DH-01"} 23.63
  ```

#### `bdx_humidity`
- **Type**: Gauge
- **Description**: Current relative humidity percentage
- **Labels**:
  - `name`: Sensor identifier
- **Example**:
  ```
  bdx_humidity{name="CGK3A-EMS-1.04-TH-DH-01"} 70.18
  ```

### CDU Metrics

#### `bdx_cdu`
- **Type**: Gauge
- **Description**: CDU metrics including alarms and parameters
- **Labels**:
  - `item`: Metric item name
  - `metrix_type`: Unit of measurement
  - `name`: CDU identifier
  - `status`: Status value
  - `type`: Metric type (alarm/parameter)
- **Example**:
  ```
  bdx_cdu{item="Average_Sec_Diff_Press",metrix_type="bar",name="CDU_1.1",status="normal",type="parameter"} 1.63
  bdx_cdu{item="CDU_1.1_Data_Hall",metrix_type="",name="CDU_1.1",status="normal",type="alarm"} 1
  ```

### Liquid Cooling Metrics

#### `bdx_liquid`
- **Type**: Gauge
- **Description**: CDU liquid cooling metrics
- **Labels**:
  - `name`: CDU identifier (e.g., "CDU_1.1")
  - `type`: Metric type (e.g., "status", "fws_flow", "fws_temp_sup")
  - `metrix_type`: Unit (e.g., "percentage", "l/min", "C")
- **Example**:
  ```
  bdx_liquid{name="CDU_1.1", type="status", metrix_type="percentage"} 0.00
  bdx_liquid{name="CDU_1.1", type="fws_flow", metrix_type="l/min"} 566.00
  bdx_liquid{name="CDU_1.1", type="fws_temp_sup", metrix_type="C"} 27.10
  ```

#### `bdx_liquid_rack`
- **Type**: Gauge
- **Description**: Rack liquid cooling metrics
- **Labels**:
  - `name`: Rack number (e.g., "7", "8")
  - `type`: Metric type (e.g., "rack_liquid_cooling", "tcs_flow", "tcs_delta_temp")
  - `metrix_type`: Unit (e.g., "kW", "l/min", "C")
- **Example**:
  ```
  bdx_liquid_rack{name="7", type="rack_liquid_cooling", metrix_type="kW"} 55.10
  bdx_liquid_rack{name="7", type="tcs_flow", metrix_type="l/min"} 148.20
  bdx_liquid_rack{name="7", type="tcs_delta_temp", metrix_type="C"} 5.4
  ```

## Deployment Guide

### Docker Compose

Create a `docker-compose.yml`:

```yaml
version: '3.8'
services:
  bdx-exporter:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - SCRAPE_INTERVAL=30s
      - SESS_MAP=your_session_map
      - PHPSESSID=your_session_id
    restart: unless-stopped
```

### Kubernetes

Example deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bdx-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: bdx-exporter
  template:
    metadata:
      labels:
        app: bdx-exporter
    spec:
      containers:
      - name: bdx-exporter
        image: bdx-exporter:latest
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        - name: SCRAPE_INTERVAL
          value: "30s"
        - name: SESS_MAP
          valueFrom:
            secretKeyRef:
              name: bdx-secrets
              key: sess_map
        - name: PHPSESSID
          valueFrom:
            secretKeyRef:
              name: bdx-secrets
              key: phpsessid
---
apiVersion: v1
kind: Service
metadata:
  name: bdx-exporter
spec:
  selector:
    app: bdx-exporter
  ports:
    - port: 8080
      targetPort: 8080
  type: ClusterIP
```

### Systemd Service

Create `/etc/systemd/system/bdx-exporter.service`:

```ini
[Unit]
Description=BDX Exporter
After=network.target

[Service]
Type=simple
User=bdx-exporter
WorkingDirectory=/opt/bdx-exporter
ExecStart=/opt/bdx-exporter/bdx-exporter
Restart=always
EnvironmentFile=/opt/bdx-exporter/.env

[Install]
WantedBy=multi-user.target
```

## Contributing Guidelines

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Clone and setup
git clone https://github.com/reski-rukmantiyo/bdx-prometheus.git
cd bdx-prometheus
go mod download

# Run tests
go test ./...

# Build
go build .

# Run with development config
export SCRAPE_INTERVAL=5s
./bdx-exporter
```

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for complex logic
- Write tests for new functionality

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Ensure session cookies are valid and not expired
   - Check that SESS_MAP and PHPSESSID are correctly set

2. **Scraping Timeouts**
   - Increase SCRAPE_TIMEOUT and HTTP_TIMEOUT values
   - Check network connectivity to BDX endpoints

3. **Missing Metrics**
   - Verify that dashboard URLs are accessible
   - Check logs for scraping errors
   - Ensure proper authentication

4. **High Memory Usage**
   - chromedp (used for scraping) requires headless browser resources
   - Consider increasing container memory limits

### Logging

The exporter logs to stdout. Use the following to view logs:

```bash
# Docker logs
docker logs <container_id>

# Kubernetes logs
kubectl logs -f deployment/bdx-exporter
```

### Monitoring the Exporter

Monitor the exporter itself using the `/health` endpoint and standard Prometheus metrics like `go_gc_duration_seconds` and `go_memstats_alloc_bytes`.