package config

import (
	"os"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Port           string
	ScrapeInterval time.Duration
	TRHURL         string
	CDUURL         string
	SessMap        string
	PHPSessID      string
	Referer        string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	port := getEnv("PORT", "8080")
	scrapeIntervalStr := getEnv("SCRAPE_INTERVAL", "30s")
	scrapeInterval, err := time.ParseDuration(scrapeIntervalStr)
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:           port,
		ScrapeInterval: scrapeInterval,
		TRHURL:         getEnv("TRH_URL", "https://app.managed360view.com/360view/trh_monitoring_dashboard.php"),
		CDUURL:         getEnv("CDU_URL", "https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38329"),
		SessMap:        getEnv("SESS_MAP", "rcbqfqyrbtqtweyxzrsasyxfcfcssacawexwqaesxxdefbxvzyaydxrwyqxvvzrufbtdeauexytusqzewzddadqaadcrrabcftrftttbdyttusascfqzqsfcrqevytucbctrdtaxqwqyfuqcavzvfwzrswyszwwytyfswvqwazaxdedq"),
		PHPSessID:      getEnv("PHPSESSID", "ghv6gfuhing3knheq9hbnvaqh5"),
		Referer:        getEnv("REFERER", "https://app.managed360view.com/360view/trh_monitoring_dashboard.php"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}