package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/reski-rukmantiyo/bdx-parser-prometheus/config"
	"github.com/reski-rukmantiyo/bdx-parser-prometheus/scraper"
)

var (
	temperatureGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_temperature",
		Help: "Current temperature reading in Celsius",
	}, []string{"name"})

	humidityGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_humidity",
		Help: "Current relative humidity percentage",
	}, []string{"name"})

	cduAlarmGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_cdu_alarm_status",
		Help: "CDU alarm status",
	}, []string{"alarm"})

	cduParameterGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_cdu_parameters",
		Help: "CDU operational parameters",
	}, []string{"parameter"})
)

// SensorData represents the sensor data from the API
type SensorData struct {
	Label string      `json:"label"`
	Temp  interface{} `json:"temp"`
	RH    interface{} `json:"rh"`
}

// Collector holds the configuration and HTTP client
type Collector struct {
	config *config.Config
	client *http.Client
}

// parseValue converts interface{} to float64, handling string and float64 types
func parseValue(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// NewCollector creates a new collector
func NewCollector(cfg *config.Config) *Collector {
	return &Collector{
		config: cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Collect collects data from all sources
func (c *Collector) Collect() {
	log.Println("Starting data collection cycle")

	// Collect temperature and humidity
	if err := c.collectTRH(); err != nil {
		log.Printf("Failed to collect TRH data: %v", err)
	} else {
		log.Println("Successfully collected TRH data")
	}

	// Collect CDU data
	if err := c.collectCDU(); err != nil {
		log.Printf("Failed to collect CDU data: %v", err)
	} else {
		log.Println("Successfully collected CDU data")
	}

	log.Println("Data collection cycle completed")
}

// collectTRH collects temperature and humidity data
func (c *Collector) collectTRH() error {
	req, err := http.NewRequest("POST", c.config.TRHURL, bytes.NewBufferString("action=inf"))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.config.Referer)
	req.Header.Set("Cookie", fmt.Sprintf("sess_map=%s; PHPSESSID=%s", c.config.SessMap, c.config.PHPSessID))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var sensors []SensorData
	if err := json.Unmarshal(body, &sensors); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Reset gauges before setting new values
	temperatureGauge.Reset()
	humidityGauge.Reset()

	for _, sensor := range sensors {
		// Convert temperature to float64
		temp, err := parseValue(sensor.Temp)
		if err != nil {
			log.Printf("Error parsing temperature for sensor %s: %v", sensor.Label, err)
			continue
		}

		// Convert humidity to float64
		humidity, err := parseValue(sensor.RH)
		if err != nil {
			log.Printf("Error parsing humidity for sensor %s: %v", sensor.Label, err)
			continue
		}

		// Set metrics with sensor name as label
		temperatureGauge.WithLabelValues(sensor.Label).Set(temp)
		humidityGauge.WithLabelValues(sensor.Label).Set(humidity)

		log.Printf("Sensor %s: temp=%.2f°C, humidity=%.2f%%", sensor.Label, temp, humidity)
	}

	log.Printf("Collected TRH data for %d sensors", len(sensors))
	return nil
}

// collectCDU collects CDU data using scraper
func (c *Collector) collectCDU() error {
	alarmData, paramData, err := scraper.ScrapeCDU(c.config.CDUURL, c.config.SessMap, c.config.PHPSessID)
	if err != nil {
		return fmt.Errorf("failed to scrape CDU data: %w", err)
	}

	// Reset gauges
	cduAlarmGauge.Reset()
	cduParameterGauge.Reset()

	// Set alarm data
	alarmCount := 0
	for key, value := range alarmData {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error parsing alarm value for %s (%s): %v", key, value, err)
			continue
		}
		cduAlarmGauge.WithLabelValues(key).Set(val)
		alarmCount++
		log.Printf("CDU Alarm - %s: %s", key, value)
	}

	// Set parameter data
	paramCount := 0
	for key, value := range paramData {
		// Clean value, remove units if any
		cleanValue := strings.Fields(value)[0]
		val, err := strconv.ParseFloat(cleanValue, 64)
		if err != nil {
			log.Printf("Error parsing parameter value for %s (%s): %v", key, cleanValue, err)
			continue
		}
		cduParameterGauge.WithLabelValues(key).Set(val)
		paramCount++
		log.Printf("CDU Parameter - %s: %s", key, value)
	}

	log.Printf("Collected CDU data: %d alarms, %d parameters", alarmCount, paramCount)
	return nil
}
