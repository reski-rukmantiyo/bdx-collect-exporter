package scraper

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// CDUAlarm represents an alarm entry
type CDUAlarm struct {
	Item   string
	Status string
}

// CDUParameter represents a parameter entry
type CDUParameter struct {
	Item  string
	Value float64
	Unit  string
}

// ScrapeCDU scrapes CDU data from the dashboard
func ScrapeCDU(url, sessMap, phpSessID string) (string, []CDUAlarm, []CDUParameter, error) {
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
		return "", nil, nil, fmt.Errorf("failed to set cookies: %v", err)
	}

	var pageHTML string

	// Run tasks
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`table`, chromedp.ByQuery), // Wait for tables to load
		chromedp.Sleep(2*time.Second), // Additional wait
		chromedp.OuterHTML("html", &pageHTML),
	)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to scrape: %v", err)
	}

	name, alarms, params := parseCDUHTML(pageHTML)

	return name, alarms, params, nil
}

// parseCDUHTML parses the full HTML and extracts name, alarms and parameters
func parseCDUHTML(html string) (string, []CDUAlarm, []CDUParameter) {
	var name string
	var alarms []CDUAlarm
	var params []CDUParameter

	// Extract name from title
	nameStart := strings.Index(html, `<h5 class="card-title mb-0">`)
	if nameStart != -1 {
		nameEnd := strings.Index(html[nameStart:], "</h5>")
		if nameEnd != -1 {
			nameText := html[nameStart+len(`<h5 class="card-title mb-0">`):nameStart+nameEnd]
			name = strings.TrimSpace(nameText)
			// Replace - with _ for Prometheus
			name = strings.ReplaceAll(name, "-", "_")
		}
	}
	if name == "" {
		name = "CDU_1.1" // fallback
	}

	// Find the alarm table: look for the table after "ALARM" header
	alarmTableStart := strings.Index(html, "ALARM")
	if alarmTableStart == -1 {
		return name, alarms, params
	}

	// Find the tbody after ALARM
	alarmTbodyStart := strings.Index(html[alarmTableStart:], "<tbody>")
	if alarmTbodyStart == -1 {
		return name, alarms, params
	}
	alarmTbodyStart += alarmTableStart

	alarmTbodyEnd := strings.Index(html[alarmTbodyStart:], "</tbody>")
	if alarmTbodyEnd == -1 {
		return name, alarms, params
	}
	alarmTbodyEnd += alarmTbodyStart

	alarmTbody := html[alarmTbodyStart:alarmTbodyEnd]

	// Parse alarm rows
	alarmRows := strings.Split(alarmTbody, "<tr>")
	for _, row := range alarmRows {
		if strings.Contains(row, "<td") && strings.Contains(row, "td-detail") {
			cells := strings.Split(row, "<td")
			if len(cells) >= 3 {
				item := extractText(cells[1])
				status := extractText(cells[2])
				if item != "" && status != "" {
					alarms = append(alarms, CDUAlarm{Item: item, Status: status})
				}
			}
		}
	}

	// Find the parameter table: look for the table after "PARAMETER" header
	paramTableStart := strings.Index(html, "PARAMETER")
	if paramTableStart == -1 {
		return name, alarms, params
	}

	// Find the tbody after PARAMETER
	paramTbodyStart := strings.Index(html[paramTableStart:], "<tbody>")
	if paramTbodyStart == -1 {
		return name, alarms, params
	}
	paramTbodyStart += paramTableStart

	paramTbodyEnd := strings.Index(html[paramTbodyStart:], "</tbody>")
	if paramTbodyEnd == -1 {
		return name, alarms, params
	}
	paramTbodyEnd += paramTbodyStart

	paramTbody := html[paramTbodyStart:paramTbodyEnd]

	// Parse parameter rows
	paramRows := strings.Split(paramTbody, "<tr>")
	for _, row := range paramRows {
		if strings.Contains(row, "<td") && strings.Contains(row, "td-detail") {
			cells := strings.Split(row, "<td")
			if len(cells) >= 4 {
				item := extractText(cells[1])
				valueStr := extractText(cells[2])
				unit := extractText(cells[3])
				if item != "" && valueStr != "" {
					value, err := strconv.ParseFloat(valueStr, 64)
					if err == nil {
						params = append(params, CDUParameter{Item: item, Value: value, Unit: unit})
					}
				}
			}
		}
	}

	return name, alarms, params
}

// extractText extracts text from HTML cell
func extractText(cell string) string {
	// Remove HTML tags and attributes
	start := strings.Index(cell, ">")
	if start == -1 {
		return ""
	}
	text := cell[start+1:]
	text = strings.ReplaceAll(text, "</td>", "")
	text = strings.ReplaceAll(text, "</th>", "")
	text = strings.ReplaceAll(text, "<b>", "")
	text = strings.ReplaceAll(text, "</b>", "")
	text = strings.TrimSpace(text)
	return text
}