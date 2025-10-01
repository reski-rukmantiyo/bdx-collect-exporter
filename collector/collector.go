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

	cduGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_cdu",
		Help: "CDU metrics including alarms and parameters",
	}, []string{"name", "type", "item", "status", "metrix_type"})

	liquidGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_liquid",
		Help: "Liquid cooling CDU metrics",
	}, []string{"name", "type", "metrix_type"})

	liquidRackGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_liquid_rack",
		Help: "Liquid cooling rack metrics",
	}, []string{"name", "type", "metrix_type"})
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

	// Collect liquid cooling data
	if err := c.collectLiquid(); err != nil {
		log.Printf("Failed to collect liquid data: %v", err)
	} else {
		log.Println("Successfully collected liquid data")
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

// collectCDU collects CDU data using scraper for multiple URLs
func (c *Collector) collectCDU() error {
	// Reset gauge
	cduGauge.Reset()

	totalAlarms := 0
	totalParams := 0
	successfulScrapes := 0

	for _, url := range c.config.CDUURLs {
		name, alarms, params, err := scraper.ScrapeCDU(url, c.config.SessMap, c.config.PHPSessID)
		if err != nil {
			log.Printf("Failed to scrape CDU data from %s: %v", url, err)
			continue
		}

		// Set alarm data
		alarmCount := 0
		for _, alarm := range alarms {
			// Normalize item name for Prometheus
			item := strings.ReplaceAll(alarm.Item, " ", "_")
			item = strings.ReplaceAll(item, "-", "_")
			status := strings.ToLower(alarm.Status)
			cduGauge.WithLabelValues(name, "alarm", item, status, "").Set(1)
			alarmCount++
			log.Printf("CDU Alarm - %s (%s): %s (%s)", name, alarm.Item, alarm.Status, status)
		}

		// Set parameter data
		paramCount := 0
		for _, param := range params {
			// Normalize item name
			item := strings.ReplaceAll(param.Item, " ", "_")
			item = strings.ReplaceAll(item, "-", "_")
			// Normalize unit
			unit := strings.ToLower(param.Unit)
			if unit == "°c" {
				unit = "celsius"
			} else if unit == "%rh" {
				unit = "percent_rh"
			}
			cduGauge.WithLabelValues(name, "parameter", item, "normal", unit).Set(param.Value)
			paramCount++
			log.Printf("CDU Parameter - %s (%s): %.2f %s", name, param.Item, param.Value, param.Unit)
		}

		totalAlarms += alarmCount
		totalParams += paramCount
		successfulScrapes++
		log.Printf("Collected CDU data for %s: %d alarms, %d parameters", name, alarmCount, paramCount)
	}

	if successfulScrapes == 0 {
		return fmt.Errorf("failed to scrape any CDU data")
	}

	log.Printf("Total CDU data collected: %d successful scrapes, %d alarms, %d parameters", successfulScrapes, totalAlarms, totalParams)
	return nil
}

// collectLiquid collects liquid cooling data
func (c *Collector) collectLiquid() error {
	// Reset gauges
	liquidGauge.Reset()
	liquidRackGauge.Reset()

	cdus, racks, err := scraper.ScrapeLiquid(c.config.LiquidURL, c.config.SessMap, c.config.PHPSessID)
	if err != nil {
		return fmt.Errorf("failed to scrape liquid data: %w", err)
	}

	// Set CDU metrics
	for _, cdu := range cdus {
		liquidGauge.WithLabelValues(cdu.Name, "status", "percentage").Set(cdu.Status)
		liquidGauge.WithLabelValues(cdu.Name, "fws_flow", "l/min").Set(cdu.FWSFlow)
		liquidGauge.WithLabelValues(cdu.Name, "fws_temp_sup", "C").Set(cdu.FWSTempSup)
		liquidGauge.WithLabelValues(cdu.Name, "fws_temp_ret", "C").Set(cdu.FWSTempRet)
		liquidGauge.WithLabelValues(cdu.Name, "tcs_flow", "l/min").Set(cdu.TCSFlow)
		liquidGauge.WithLabelValues(cdu.Name, "tcs_temp_sup", "C").Set(cdu.TCSTempSup)
		liquidGauge.WithLabelValues(cdu.Name, "tcs_temp_ret", "C").Set(cdu.TCSTempRet)
		log.Printf("Liquid CDU %s: status=%.2f%%, fws_flow=%.2f l/min, fws_temp_sup=%.2f°C, fws_temp_ret=%.2f°C, tcs_flow=%.2f l/min, tcs_temp_sup=%.2f°C, tcs_temp_ret=%.2f°C", cdu.Name, cdu.Status, cdu.FWSFlow, cdu.FWSTempSup, cdu.FWSTempRet, cdu.TCSFlow, cdu.TCSTempSup, cdu.TCSTempRet)
	}

	// Set rack metrics
	for _, rack := range racks {
		liquidRackGauge.WithLabelValues(rack.RackNumber, "rack_liquid_cooling", "kW").Set(rack.RackLiquidCooling)
		liquidRackGauge.WithLabelValues(rack.RackNumber, "tcs_flow", "l/min").Set(rack.TCSFlow)
		liquidRackGauge.WithLabelValues(rack.RackNumber, "tcs_delta_temp", "C").Set(rack.TCSDeltaTemp)
		liquidRackGauge.WithLabelValues(rack.RackNumber, "tcs_temp_supply", "C").Set(rack.TCSTempSupply)
		log.Printf("Liquid Rack %s: rack_liquid_cooling=%.2f kW, tcs_flow=%.2f l/min, tcs_delta_temp=%.2f°C, tcs_temp_supply=%.2f°C", rack.RackNumber, rack.RackLiquidCooling, rack.TCSFlow, rack.TCSDeltaTemp, rack.TCSTempSupply)
	}

	log.Printf("Collected liquid data: %d CDUs, %d racks", len(cdus), len(racks))
	return nil
}
