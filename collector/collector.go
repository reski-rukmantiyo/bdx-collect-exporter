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
	temperatureGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bdx_temperature_celsius",
		Help: "Current temperature reading in Celsius",
	})

	humidityGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bdx_humidity_percent",
		Help: "Current humidity percentage",
	})

	cduAlarmGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_cdu_alarm_status",
		Help: "CDU alarm status",
	}, []string{"alarm"})

	cduParameterGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bdx_cdu_parameters",
		Help: "CDU operational parameters",
	}, []string{"parameter"})
)

// TRHData represents the temperature and humidity data
type TRHData struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

// Collector holds the configuration and HTTP client
type Collector struct {
	config *config.Config
	client *http.Client
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
	// Collect temperature and humidity
	if err := c.collectTRH(); err != nil {
		log.Printf("Error collecting TRH data: %v", err)
	}

	// Collect CDU data
	if err := c.collectCDU(); err != nil {
		log.Printf("Error collecting CDU data: %v", err)
	}
}

// collectTRH collects temperature and humidity data
func (c *Collector) collectTRH() error {
	req, err := http.NewRequest("POST", c.config.TRHURL, bytes.NewBufferString("action=inf"))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.config.Referer)
	req.Header.Set("Cookie", fmt.Sprintf("sess_map=%s; PHPSESSID=%s", c.config.SessMap, c.config.PHPSessID))

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data TRHData
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	temperatureGauge.Set(data.Temperature)
	humidityGauge.Set(data.Humidity)

	log.Printf("Collected TRH: temp=%.2f, humidity=%.2f", data.Temperature, data.Humidity)
	return nil
}

// collectCDU collects CDU data using scraper
func (c *Collector) collectCDU() error {
	alarmData, paramData, err := scraper.ScrapeCDU(c.config.CDUURL, c.config.SessMap, c.config.PHPSessID)
	if err != nil {
		return err
	}

	// Reset gauges
	cduAlarmGauge.Reset()
	cduParameterGauge.Reset()

	// Set alarm data
	for key, value := range alarmData {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error parsing alarm value %s: %v", value, err)
			continue
		}
		cduAlarmGauge.WithLabelValues(key).Set(val)
	}

	// Set parameter data
	for key, value := range paramData {
		// Clean value, remove units if any
		cleanValue := strings.Fields(value)[0]
		val, err := strconv.ParseFloat(cleanValue, 64)
		if err != nil {
			log.Printf("Error parsing parameter value %s: %v", cleanValue, err)
			continue
		}
		cduParameterGauge.WithLabelValues(key).Set(val)
	}

	log.Printf("Collected CDU: alarms=%d, params=%d", len(alarmData), len(paramData))
	return nil
}