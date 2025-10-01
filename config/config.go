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
	LiquidURL      string
	CDUURLs        []string
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
		LiquidURL:      getEnv("LIQUID_URL", "https://app.managed360view.com/360view/liquid_cooling_overview.php"),
		CDUURLs: []string{
			"https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38329", // CDU 1.1
			"https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38337", // CDU 1.2
			"https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38331", // CDU 2.1
			"https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38339", // CDU 2.2
			"https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38333", // CDU 3.1
			"https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38341", // CDU 3.2
			"https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38335", // CDU 4.1
			"https://app.managed360view.com/360view/cdu_dashboard.php?cabinetid=38343", // CDU 4.2
		},
		SessMap:   getEnv("SESS_MAP", "rcbqfqyrbtqtweyxzrsasyxfcfcssacawexwqaesxxdefbxvzyaydxrwyqxvvzrufbtdeauexytusqzewzddadqaadcrrabcftrftttbdyttusascfqzqsfcrqevytucbctrdtaxqwqyfuqcavzvfwzrswyszwwytyfswvqwazaxdedq"),
		PHPSessID: getEnv("PHPSESSID", "ghv6gfuhing3knheq9hbnvaqh5"),
		Referer:   getEnv("REFERER", "https://app.managed360view.com/360view/trh_monitoring_dashboard.php"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
