package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ScrapeCDU scrapes CDU data from the dashboard
func ScrapeCDU(url, sessMap, phpSessID string) (map[string]string, map[string]string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create chromedp context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	// Set cookies
	cookies := []*network.CookieParam{
		{
			Name:   "sess_map",
			Value:  sessMap,
			Domain: "app.managed360view.com",
			Path:   "/",
		},
		{
			Name:   "PHPSESSID",
			Value:  phpSessID,
			Domain: "app.managed360view.com",
			Path:   "/",
		},
	}

	if err := chromedp.Run(taskCtx, network.SetCookies(cookies)); err != nil {
		return nil, nil, fmt.Errorf("failed to set cookies: %v", err)
	}

	var alarmHTML, paramHTML string

	// Run tasks
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`table`, chromedp.ByQuery), // Wait for tables to load
		chromedp.Sleep(2*time.Second), // Additional wait
		chromedp.Evaluate(`document.querySelectorAll('table')[0].outerHTML`, &alarmHTML),
		chromedp.Evaluate(`document.querySelectorAll('table')[1].outerHTML`, &paramHTML),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to scrape: %v", err)
	}

	alarmData := parseTable(alarmHTML)
	paramData := parseTable(paramHTML)

	return alarmData, paramData, nil
}

// parseTable parses HTML table and returns key-value map
func parseTable(html string) map[string]string {
	data := make(map[string]string)

	// Simple parsing: split by <tr>, then <td>
	rows := strings.Split(html, "<tr>")
	for _, row := range rows {
		if strings.Contains(row, "<td>") {
			cells := strings.Split(row, "<td>")
			if len(cells) >= 3 {
				key := extractText(cells[1])
				value := extractText(cells[2])
				if key != "" && value != "" {
					data[key] = value
				}
			}
		}
	}

	return data
}

// extractText extracts text from HTML cell
func extractText(cell string) string {
	// Remove HTML tags
	text := strings.ReplaceAll(cell, "<br>", " ")
	text = strings.ReplaceAll(text, "</td>", "")
	text = strings.ReplaceAll(text, "</th>", "")
	text = strings.TrimSpace(text)
	return text
}