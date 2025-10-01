package config

import (
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Port             string
	ScrapeInterval   time.Duration
	HTTPTimeout      time.Duration
	ScrapeTimeout    time.Duration
	TRHURL           string
	LiquidCoolingURL string
	CDUURLs          []string
	SessMap          string
	PHPSessID        string
	Referer          string
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	scrapeIntervalStr := getEnv("SCRAPE_INTERVAL", "30s")
	scrapeInterval, err := time.ParseDuration(scrapeIntervalStr)
	if err != nil {
		return nil, err
	}

	httpTimeoutStr := getEnv("HTTP_TIMEOUT", "10s")
	httpTimeout, err := time.ParseDuration(httpTimeoutStr)
	if err != nil {
		return nil, err
	}

	scrapeTimeoutStr := getEnv("SCRAPE_TIMEOUT", "30s")
	scrapeTimeout, err := time.ParseDuration(scrapeTimeoutStr)
	if err != nil {
		return nil, err
	}

	cduURLsStr := getEnv("CDU_URLS", "https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38329,https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38337,https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38331,https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38339,https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38333,https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38341,https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38335,https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38343")
	var cduURLs []string
	if cduURLsStr != "" {
		cduURLs = strings.Split(cduURLsStr, ",")
		for i := range cduURLs {
			cduURLs[i] = strings.TrimSpace(cduURLs[i])
		}
	}

	return &Config{
		Port:             port,
		ScrapeInterval:   scrapeInterval,
		HTTPTimeout:      httpTimeout,
		ScrapeTimeout:    scrapeTimeout,
		TRHURL:           getEnv("TRH_URL", "https://app.managed360view.com/360view/trh_monitoring_dashboard.php"),
		LiquidCoolingURL: getEnv("LIQUID_URL", "https://app.managed360view.com/360view/liquid_cooling_overview.php"),
		CDUURLs:          cduURLs,
		SessMap:          getEnv("SESS_MAP", "rcbqfqyrbtqtweyxzrsasyxfcfcssacawexwqaesxxdefbxvzyaydxrwyqxvvzrufbtdeauexytusqzewzddadqaadcrrabcftrftttbdyttusascfqzqsfcrqevytucbctrdtaxqwqyfuqcavzvfwzrswyszwwytyfswvqwazaxdedq"),
		PHPSessID:        getEnv("PHPSESSID", "ghv6gfuhing3knheq9hbnvaqh5"),
		Referer:          getEnv("REFERER", "https://app.managed360view.com/360view/trh_monitoring_dashboard.php"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
