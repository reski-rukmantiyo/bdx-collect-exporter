package scraper

import (
	"context"
	"fmt"
	"regexp"
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

// LiquidCDU represents CDU liquid cooling data
type LiquidCDU struct {
	Name       string
	Status     float64
	FWSFlow    float64
	FWSTempSup float64
	FWSTempRet float64
	TCSFlow    float64
	TCSTempSup float64
	TCSTempRet float64
}

// LiquidRack represents rack liquid cooling data
type LiquidRack struct {
	RackNumber         string
	RackLiquidCooling  float64
	TCSFlow            float64
	TCSDeltaTemp       float64
	TCSTempSupply      float64
}

// ScrapeCDU scrapes CDU data from the dashboard
func ScrapeCDU(url, sessMap, phpSessID string, timeout time.Duration) (string, []CDUAlarm, []CDUParameter, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
				item := normalizeItem(extractText(cells[1]))
				status := strings.ToLower(extractText(cells[2]))
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
				item := normalizeItem(extractText(cells[1]))
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

// ScrapeLiquidCooling scrapes liquid cooling data from the overview page
func ScrapeLiquidCooling(url, sessMap, phpSessID string, timeout time.Duration) ([]LiquidCDU, []LiquidRack, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

	var pageHTML string

	// Run tasks
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`table`, chromedp.ByQuery), // Wait for tables to load
		chromedp.Sleep(2*time.Second), // Additional wait
		chromedp.OuterHTML("html", &pageHTML),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to scrape: %v", err)
	}

	cdus, racks := parseLiquidHTML(pageHTML)

	return cdus, racks, nil
}

// parseLiquidHTML parses the liquid cooling HTML and extracts CDU and rack data
func parseLiquidHTML(html string) ([]LiquidCDU, []LiquidRack) {
	var cdus []LiquidCDU
	var racks []LiquidRack

	// Parse CDU tables
	// Look for tables with "CGK3A-CL-1.04-CDU-" in the header
	cduPattern := `CGK3A-CL-1\.04-CDU-(\d+\.\d+) STATUS`
	cduRegex := regexp.MustCompile(cduPattern)
	matches := cduRegex.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		cduName := "CDU_" + match[1]

		// Find the table start after the header
		headerIndex := strings.Index(html, match[0])
		if headerIndex == -1 {
			continue
		}

		// Find the table after the header
		tableStart := strings.Index(html[headerIndex:], "<table")
		if tableStart == -1 {
			continue
		}
		tableStart += headerIndex

		tableEnd := strings.Index(html[tableStart:], "</table>")
		if tableEnd == -1 {
			continue
		}
		tableEnd += tableStart

		tableHTML := html[tableStart:tableEnd]

		cdu := parseCDUTable(tableHTML, cduName)
		if cdu.Name != "" {
			cdus = append(cdus, cdu)
		}
	}

	// Parse rack tables
	// Look for "ENERGY VALVE STATUS COMPARTMENT" tables
	rackPattern := `ENERGY VALVE STATUS COMPARTMENT ([A-Z]+)`
	rackRegex := regexp.MustCompile(rackPattern)
	rackMatches := rackRegex.FindAllStringSubmatch(html, -1)

	for _, match := range rackMatches {
		if len(match) < 2 {
			continue
		}
		compartment := match[1]

		// Find the table start after the header
		headerIndex := strings.Index(html, match[0])
		if headerIndex == -1 {
			continue
		}

		// Find the table after the header
		tableStart := strings.Index(html[headerIndex:], "<table")
		if tableStart == -1 {
			continue
		}
		tableStart += headerIndex

		tableEnd := strings.Index(html[tableStart:], "</table>")
		if tableEnd == -1 {
			continue
		}
		tableEnd += tableStart

		tableHTML := html[tableStart:tableEnd]

		rackData := parseRackTable(tableHTML, compartment)
		racks = append(racks, rackData...)
	}

	return cdus, racks
}

// parseCDUTable parses a single CDU table
func parseCDUTable(tableHTML, cduName string) LiquidCDU {
	var cdu LiquidCDU
	cdu.Name = cduName

	// Find all <tr> rows
	rows := strings.Split(tableHTML, "<tr")
	for _, row := range rows {
		if !strings.Contains(row, "<td") {
			continue
		}

		// Split by <td
		cells := strings.Split(row, "<td")
		if len(cells) < 3 {
			continue
		}

		// Extract label-value pairs
		for i := 1; i < len(cells); i += 2 {
			if i+1 >= len(cells) {
				break
			}
			label := extractText(cells[i])
			valueStr := extractText(cells[i+1])

			if label == "" || valueStr == "" {
				continue
			}

			// Normalize units
			valueStr = strings.ReplaceAll(valueStr, "I/min", "l/min")
			valueStr = strings.ReplaceAll(valueStr, "°C", "C")

			value, err := strconv.ParseFloat(strings.Fields(valueStr)[0], 64)
			if err != nil {
				continue
			}

			switch strings.ToLower(strings.ReplaceAll(label, " ", "_")) {
			case "cdu_cooling":
				cdu.Status = value
			case "fws_flow":
				cdu.FWSFlow = value
			case "fws_temp_sup":
				cdu.FWSTempSup = value
			case "fws_temp_ret":
				cdu.FWSTempRet = value
			case "tcs_flow":
				cdu.TCSFlow = value
			case "tcs_temp_sup":
				cdu.TCSTempSup = value
			case "tcs_temp_ret":
				cdu.TCSTempRet = value
			}
		}
	}

	return cdu
}

// parseRackTable parses a single rack table
func parseRackTable(tableHTML, compartment string) []LiquidRack {
	var racks []LiquidRack

	// Find the header row to get rack numbers
	headerStart := strings.Index(tableHTML, "<thead")
	if headerStart == -1 {
		return racks
	}
	headerEnd := strings.Index(tableHTML[headerStart:], "</thead>")
	if headerEnd == -1 {
		return racks
	}
	headerEnd += headerStart
	headerHTML := tableHTML[headerStart:headerEnd]

	// Extract rack numbers from header
	var rackNumbers []string
	thMatches := regexp.MustCompile(`<th[^>]*>([^<]+)</th>`).FindAllStringSubmatch(headerHTML, -1)
	for _, match := range thMatches {
		if len(match) > 1 && strings.Contains(match[1], "RACK ") {
			rackNum := strings.TrimSpace(strings.ReplaceAll(match[1], "RACK ", ""))
			rackNumbers = append(rackNumbers, rackNum)
		}
	}

	// Find tbody
	tbodyStart := strings.Index(tableHTML, "<tbody")
	if tbodyStart == -1 {
		return racks
	}
	tbodyEnd := strings.Index(tableHTML[tbodyStart:], "</tbody>")
	if tbodyEnd == -1 {
		return racks
	}
	tbodyEnd += tbodyStart
	tbodyHTML := tableHTML[tbodyStart:tbodyEnd]

	// Parse rows
	rows := strings.Split(tbodyHTML, "<tr")
	for _, row := range rows {
		if !strings.Contains(row, "<td") {
			continue
		}

		cells := strings.Split(row, "<td")
		if len(cells) < 2 {
			continue
		}

		label := extractText(cells[1])
		label = strings.ToLower(strings.ReplaceAll(label, " ", "_"))

		// Skip if not a data row
		if label == "" {
			continue
		}

		// Extract values for each rack
		for i, rackNum := range rackNumbers {
			if i+2 >= len(cells) {
				continue
			}
			valueStr := extractText(cells[i+2])

			// Normalize units
			valueStr = strings.ReplaceAll(valueStr, "I/min", "l/min")
			valueStr = strings.ReplaceAll(valueStr, "°C", "C")
			valueStr = strings.ReplaceAll(valueStr, "kW", "kW")

			value, err := strconv.ParseFloat(strings.Fields(valueStr)[0], 64)
			if err != nil {
				continue
			}

			// Find or create rack
			var rack *LiquidRack
			for j := range racks {
				if racks[j].RackNumber == rackNum {
					rack = &racks[j]
					break
				}
			}
			if rack == nil {
				racks = append(racks, LiquidRack{RackNumber: rackNum})
				rack = &racks[len(racks)-1]
			}

			switch label {
			case "rack_liquid_cooling":
				rack.RackLiquidCooling = value
			case "tcs_flow":
				rack.TCSFlow = value
			case "tcs_delta_temp":
				rack.TCSDeltaTemp = value
			case "tcs_temp_supply":
				rack.TCSTempSupply = value
			}
		}
	}

	return racks
}

// extractText extracts text from HTML cell
func extractText(cell string) string {
    // Remove HTML tags and attributes
    start := strings.Index(cell, ">")
    if start == -1 {
        return ""
    }
    text := cell[start+1:]
    // Remove all remaining HTML tags
    text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
    text = strings.TrimSpace(text)
    return text
}

// normalizeItem normalizes item names for Prometheus
func normalizeItem(item string) string {
    // Replace spaces and dashes with underscores
    item = strings.ReplaceAll(item, " ", "_")
    item = strings.ReplaceAll(item, "-", "_")
    // Replace multiple underscores with single underscore
    item = regexp.MustCompile(`_+`).ReplaceAllString(item, "_")
    // Remove leading/trailing underscores
    item = strings.Trim(item, "_")
    return item
}