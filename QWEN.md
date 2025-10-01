# BDX Exporter - Project Summary

## Key Project Information

**Project Name**: BDX Exporter  
**Purpose**: Prometheus exporter for BDX (360°View) data center monitoring system  
**Language**: Go 1.19+  
**Framework**: Gin web framework  
**Scraping**: chromedp for browser automation  
**Metrics**: Prometheus client library  

**Package**: `github.com/reski-rukmantiyo/bdx-parser-prometheus`

## Implementation Notes

### Architecture Overview

The exporter consists of three main components:

1. **Main Server** (`main.go`): HTTP server with Gin, handles `/metrics` and `/health` endpoints
2. **Collector** (`collector/collector.go`): Core logic for data collection and metric creation
3. **Configuration** (`config/config.go`): Environment-based configuration management

### Data Collection Strategy

- **Temperature/Humidity**: HTTP POST to `trh_monitoring_dashboard.php` with `action=inf`
- **CDU Status**: Browser scraping of individual CDU dashboard pages
- **Liquid Cooling**: Browser scraping of liquid cooling overview page

### Authentication

- Uses session cookies: `sess_map` and `PHPSESSID`
- Requires valid 360View login session
- Cookies must be refreshed periodically (not currently automated)

### Metric Types

- **bdx_temperature**: Gauge for temperature readings (°C)
- **bdx_humidity**: Gauge for humidity readings (%)
- **bdx_cdu**: Gauge for CDU alarms and parameters
- **bdx_liquid**: Gauge for CDU liquid cooling metrics
- **bdx_liquid_rack**: Gauge for rack-level liquid cooling metrics

## Configuration Details

### Environment Variables

All configuration is loaded from environment variables with `.env` file support via godotenv.

**Core Settings:**
- `PORT`: Server port (default: 8080)
- `SCRAPE_INTERVAL`: Collection frequency (default: 30s)
- `HTTP_TIMEOUT`: HTTP request timeout (default: 10s)
- `SCRAPE_TIMEOUT`: Scraping operation timeout (default: 30s)

**Endpoint URLs:**
- `TRH_URL`: Temperature/humidity dashboard URL
- `LIQUID_URL`: Liquid cooling overview URL
- `CDU_URLS`: Comma-separated list of CDU dashboard URLs

**Authentication:**
- `SESS_MAP`: Session map cookie value
- `PHPSESSID`: PHP session ID cookie value
- `REFERER`: HTTP referer header

### Default CDU URLs

```
https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38329 (CDU-1.1)
https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38337 (CDU-1.2)
https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38331 (CDU-2.1)
https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38339 (CDU-2.2)
https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38333 (CDU-3.1)
https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38341 (CDU-3.2)
https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38335 (CDU-4.1)
https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38343 (CDU-4.2)
```

## Special Considerations and Gotchas

### Session Management
- Session cookies expire; manual refresh required
- No automatic re-authentication implemented
- Monitor `/health` endpoint for collection failures

### Resource Usage
- chromedp requires headless Chrome/Chromium
- Higher memory usage compared to pure HTTP clients
- Consider resource limits in container orchestration

### Scraping Reliability
- Web scraping is brittle; page structure changes break collection
- Network timeouts and connectivity issues affect reliability
- Rate limiting may be required for high-frequency scraping

### Data Processing
- JSON parsing for temperature/humidity data
- HTML table parsing for CDU and liquid cooling data
- Metric naming conventions must be consistent

### Error Handling
- Graceful degradation: failed sources don't stop other collections
- Health status tracks last successful collection
- Logging for debugging scraping issues

### Deployment Considerations
- Single replica recommended (no coordination needed)
- Persistent storage not required
- Network access to BDX endpoints mandatory

## Update History

### Phase 1 Implementation
- **Temperature & Humidity**: HTTP-based collection from dashboard API
- **CDU Status**: Browser scraping of alarm and parameter tables
- **Liquid Cooling**: Browser scraping of overview dashboard
- **Metrics**: Basic Prometheus gauge metrics
- **Health Checks**: Collection status monitoring

### Known Issues
- Session cookie expiration handling
- No retry logic for failed scrapes
- Limited error reporting in metrics

### Future Enhancements
- Automatic session refresh
- Configurable retry logic
- Alert rules configuration
- Dashboard visualization
- Historical data storage
- Multiple cabinet ID support

## Development Notes

### Dependencies
- `github.com/gin-gonic/gin`: Web framework
- `github.com/chromedp/chromedp`: Browser automation
- `github.com/prometheus/client_golang`: Metrics client
- `github.com/joho/godotenv`: Environment loading

### Testing
- Unit tests for data parsing logic
- Integration tests for metric collection
- Mock responses for external dependencies

### Monitoring
- Application metrics via `/metrics`
- Health status via `/health`
- Structured logging for operations

### Security
- Store session credentials securely
- Use HTTPS for all communications
- Validate input parameters
- No sensitive data in logs

## Qwen Added Memories
- BDX Exporter is a Go-based Prometheus exporter that collects temperature, humidity, CDU status, and liquid cooling metrics from 360View data center dashboards using HTTP requests and browser scraping with chromedp.
