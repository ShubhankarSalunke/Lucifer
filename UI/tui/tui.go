package tui

import (
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"cli/api"

	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
	"github.com/rivo/tview"
)

func init() {
	for _, p := range []string{"../../.env", "../.env", ".env", "../../../.env"} {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}
}

func GetMonitorPage(app *tview.Application) *tview.Flex {
	// State
	viewMode := "live"
	aggregateWindow := "overall"
	statusTable := tview.NewTable().SetBorders(false)
	statusTable.SetBorder(true).
		SetTitle(" SYSTEM HEALTH (ACTIVE) ").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorGreen)

	setTableRow := func(row int, label, value string, color tcell.Color) {
		statusTable.SetCell(row, 0, tview.NewTableCell(label).SetTextColor(tcell.ColorGray))
		statusTable.SetCell(row, 1, tview.NewTableCell(value).SetTextColor(color).SetExpansion(1))
	}

	feed := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	feed.SetBorder(true).
		SetTitle(" LIVE TELEMETRY STREAM ").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorGreen)

	graphView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	graphView.SetBorder(true).
		SetTitle(" METRIC GRAPHS ").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorAqua)

	agentBox := tview.NewList().ShowSecondaryText(true)
	agentBox.SetBorder(true).
		SetTitle(" DISCOVERED HOST AGENTS ").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.NewRGBColor(140, 140, 160))

	agentBox.SetSelectedTextColor(tcell.ColorGreen).
		SetSelectedBackgroundColor(tcell.ColorBlack)

	var agentsData []api.Agent
	selectedAgentIdx := 0
	var refreshInFlight atomic.Bool
	var lastLoggedMetricTimestamp string
	feedLines := make([]string, 0, 120)

	appendFeedLine := func(line string) string {
		feedLines = append(feedLines, line)
		if len(feedLines) > 120 {
			feedLines = feedLines[len(feedLines)-120:]
		}
		return strings.Join(feedLines, "\n")
	}

	agentBox.SetChangedFunc(func(index int, _, _ string, _ rune) {
		selectedAgentIdx = index
	})

	sampleSeries := func(values []float64, width int) []float64 {
		if len(values) == 0 {
			return nil
		}
		if width <= 0 || len(values) <= width {
			return values
		}
		step := float64(len(values)) / float64(width)
		compressed := make([]float64, 0, width)
		for i := 0; i < width; i++ {
			idx := int(math.Floor(float64(i) * step))
			if idx >= len(values) {
				idx = len(values) - 1
			}
			compressed = append(compressed, values[idx])
		}
		return compressed
	}

	renderSparkline := func(values []float64, width int) string {
		if len(values) == 0 {
			return "[gray]no data[-]"
		}
		values = sampleSeries(values, width)

		blocks := []rune("▁▂▃▄▅▆▇█")
		minVal := values[0]
		maxVal := values[0]
		for _, v := range values[1:] {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal == minVal {
			return strings.Repeat(string(blocks[len(blocks)/2]), len(values))
		}

		var b strings.Builder
		for _, v := range values {
			ratio := (v - minVal) / (maxVal - minVal)
			idx := int(math.Round(ratio * float64(len(blocks)-1)))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(blocks) {
				idx = len(blocks) - 1
			}
			b.WriteRune(blocks[idx])
		}
		return b.String()
	}

	formatSeries := func(label string, values []float64, color string) string {
		if len(values) == 0 {
			return fmt.Sprintf("[white]%s[-] [gray]no data[-]", label)
		}

		var minVal, maxVal, latest float64
		minVal = values[0]
		maxVal = values[0]
		for _, v := range values {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
			latest = v
		}

		return fmt.Sprintf(
			"[white::b]%-12s[-] [%s]%s[-] [gray]min %.1f max %.1f now %.1f[-]",
			label,
			color,
			renderSparkline(values, 48),
			minVal,
			maxVal,
			latest,
		)
	}

	renderBarChart := func(labels []string, values []float64, colors []string, height int) string {
		if len(values) == 0 {
			return "[gray]no chart data[-]"
		}
		if height < 4 {
			height = 4
		}

		maxVal := 0.0
		for _, v := range values {
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal == 0 {
			maxVal = 1
		}

		var lines []string

		// Value Line (Above)
		var valueLine strings.Builder
		for _, v := range values {
			// Higher precision for small values to avoid "0" glitch
			if v > 0 && v < 1 {
				valueLine.WriteString(fmt.Sprintf("[white::b]%3.1f[-] ", v))
			} else {
				valueLine.WriteString(fmt.Sprintf("[white]%3.0f[-] ", v))
			}
		}
		lines = append(lines, valueLine.String())

		// Bar Rows
		for row := height; row >= 1; row-- {
			var line strings.Builder
			for i, v := range values {
				barHeight := int(math.Round((v / maxVal) * float64(height)))
				cell := "    "
				if barHeight >= row {
					color := "green"
					if i < len(colors) && colors[i] != "" {
						color = colors[i]
					}
					cell = fmt.Sprintf("[%s]███[-] ", color)
				}
				line.WriteString(cell)
			}
			lines = append(lines, line.String())
		}

		// Label Line
		var labelLine strings.Builder
		for _, label := range labels {
			labelLine.WriteString(fmt.Sprintf("[gray]%-3s[-] ", label))
		}
		lines = append(lines, labelLine.String())

		return strings.Join(lines, "\n")
	}

	renderLineChart := func(values []float64, width, height int, color string) string {
		if len(values) == 0 {
			return "[gray]no line data[-]"
		}
		if width < 8 {
			width = 8
		}
		if height < 6 {
			height = 6
		}

		values = sampleSeries(values, width)
		minVal := values[0]
		maxVal := values[0]
		for _, v := range values[1:] {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal == minVal {
			maxVal = minVal + 1
		}

		grid := make([][]string, height)
		for y := range grid {
			grid[y] = make([]string, width)
			for x := range grid[y] {
				grid[y][x] = " "
			}
		}

		for x, v := range values {
			ratio := (v - minVal) / (maxVal - minVal)
			y := height - 1 - int(math.Round(ratio*float64(height-1)))
			if y < 0 {
				y = 0
			}
			if y >= height {
				y = height - 1
			}
			grid[y][x] = fmt.Sprintf("[%s]•[-]", color)
		}

		var lines []string
		for y := 0; y < height; y++ {
			label := "     "
			if y == 0 {
				label = fmt.Sprintf("%5.1f", maxVal)
			} else if y == height/2 {
				label = fmt.Sprintf("%5.1f", (maxVal+minVal)/2)
			} else if y == height-1 {
				label = fmt.Sprintf("%5.1f", minVal)
			}

			var row strings.Builder
			for x := 0; x < width; x++ {
				row.WriteString(grid[y][x])
			}
			lines = append(lines, fmt.Sprintf("[gray]%s │[-] %s", label, row.String()))
		}
		lines = append(lines, fmt.Sprintf("[gray]%s└%s[-]", strings.Repeat(" ", 6), strings.Repeat("─", width+1)))
		return strings.Join(lines, "\n")
	}

	renderGraphDashboard := func(modeLabel string, cpuSeries, netInSeries, netOutSeries, diskSeries []float64) string {
		header := fmt.Sprintf("[green::b]Termui-Inspired %s[-]", modeLabel)
		sparkPanel := strings.Join([]string{
			header,
			"",
			fmt.Sprintf("[white]cpu  [-][aqua]%s[-]", renderSparkline(cpuSeries, 40)),
			fmt.Sprintf("[white]net  [-][red]%s[-]", renderSparkline(netInSeries, 40)),
			fmt.Sprintf("[white]disk [-][yellow]%s[-]", renderSparkline(diskSeries, 40)),
		}, "\n")

		cpuLatest := 0.0
		netInLatest := 0.0
		netOutLatest := 0.0
		diskLatest := 0.0
		if len(cpuSeries) > 0 {
			cpuLatest = cpuSeries[len(cpuSeries)-1]
		}
		if len(netInSeries) > 0 {
			netInLatest = netInSeries[len(netInSeries)-1]
		}
		if len(netOutSeries) > 0 {
			netOutLatest = netOutSeries[len(netOutSeries)-1]
		}
		if len(diskSeries) > 0 {
			diskLatest = diskSeries[len(diskSeries)-1]
		}

		barPanel := renderBarChart(
			[]string{"CPU", "IN", "OUT", "DSK"},
			[]float64{cpuLatest, netInLatest, netOutLatest, diskLatest},
			[]string{"green", "aqua", "teal", "yellow"},
			6,
		)

		linePanel := renderLineChart(cpuSeries, 42, 8, "red")
		netLinePanel := renderLineChart(append([]float64(nil), netInSeries...), 42, 8, "yellow")

		return strings.Join([]string{
			sparkPanel,
			"",
			"[green]Bar Chart[-]",
			barPanel,
			"",
			"[green]CPU Line Chart[-]",
			linePanel,
			"",
			"[green]Network Line Chart[-]",
			netLinePanel,
		}, "\n")
	}

	updateGraphs := func(history []api.ComputeSummary, title string) {
		var cpuSeries []float64
		var netSeries []float64
		var diskSeries []float64
		var netInSeries []float64
		var netOutSeries []float64
		for _, point := range history {
			cpuSeries = append(cpuSeries, point.ComputeMetric.CPUUtilization)
			netSeries = append(netSeries, (point.ComputeMetric.NetworkInBytes+point.ComputeMetric.NetworkOutBytes)/1024)
			diskSeries = append(diskSeries, (point.ComputeMetric.DiskReadBytes+point.ComputeMetric.DiskWriteBytes)/1024)
			netInSeries = append(netInSeries, point.ComputeMetric.NetworkInBytes/1024)
			netOutSeries = append(netOutSeries, point.ComputeMetric.NetworkOutBytes/1024)
		}

		graphView.SetTitle(title)
		graphView.SetText(strings.Join([]string{
			formatSeries("CPU %", cpuSeries, "green"),
			formatSeries("NET KB/s", netSeries, "yellow"),
			formatSeries("DISK KB/s", diskSeries, "aqua"),
			"",
			renderGraphDashboard(title, cpuSeries, netInSeries, netOutSeries, diskSeries),
			"",
			"[gray]Keys: l = live | n = normal | w = normal window toggle[-]",
		}, "\n"))
	}

	// Helper to refresh live data
	updateData := func() {
		if !refreshInFlight.CompareAndSwap(false, true) {
			return
		}
		defer refreshInFlight.Store(false)

		// Merge agents from both sources: management gateway + InfluxDB autodiscovery.
		seenIDs := make(map[string]bool)
		var agentsList []api.Agent

		if gwAgents, gwErr := api.GetAgents(); gwErr == nil {
			for _, a := range gwAgents {
				if !seenIDs[a.ID] {
					seenIDs[a.ID] = true
					agentsList = append(agentsList, a)
				}
			}
		}
		if disc, discErr := api.GetDiscoveredAgents(); discErr == nil {
			for _, a := range disc {
				if !seenIDs[a.ID] {
					seenIDs[a.ID] = true
					agentsList = append(agentsList, a)
				}
			}
		}

		nextSelectedIdx := selectedAgentIdx
		if nextSelectedIdx >= len(agentsList) {
			nextSelectedIdx = len(agentsList) - 1
		}
		if nextSelectedIdx < 0 {
			nextSelectedIdx = 0
		}

		var (
			metrics       *api.ComputeSummary
			aggregate     *api.ComputeAggregate
			history       []api.ComputeSummary
			statusTitle   string
			statusColor   tcell.Color
			graphTitle    string
			feedText      string
			targetAgentID string
		)

		if len(agentsList) > 0 {
			targetAgentID = agentsList[nextSelectedIdx].ID

			if viewMode == "live" {
				metrics, _ = api.GetComputeMetrics(targetAgentID)
				history, _ = api.GetComputeHistory(targetAgentID, 2*time.Minute)
				statusTitle = " SYSTEM HEALTH (LIVE) "
				statusColor = tcell.ColorGreen
				graphTitle = " LIVE METRIC GRAPHS (LAST 2M) "

				if metrics != nil && metrics.Timestamp != "" && metrics.Timestamp != lastLoggedMetricTimestamp {
					lastLoggedMetricTimestamp = metrics.Timestamp
					ts := time.Now().Format("15:04:05")
					line := fmt.Sprintf("[gray]%s[-] [green]Metric Packet[-] | CPU: [white]%d%%[-] | Net: [white]%.1fKb[-] | [gray]OK[-]",
						ts,
						int(metrics.ComputeMetric.CPUUtilization),
						(metrics.ComputeMetric.NetworkInBytes+metrics.ComputeMetric.NetworkOutBytes)/1024,
					)
					feedText = appendFeedLine(line)
				}
			} else {
				statusTitle = fmt.Sprintf(" SYSTEM HEALTH (%s AGGREGATE) ", strings.ToUpper(aggregateWindow))
				statusColor = tcell.ColorYellow
				graphTitle = fmt.Sprintf(" NORMAL MODE GRAPHS (%s) ", strings.ToUpper(aggregateWindow))
				aggregate, _ = api.GetComputeAggregate(targetAgentID, aggregateWindow)
				history, _ = api.GetComputeHistoryForScope(targetAgentID, aggregateWindow)
			}
		}

		app.QueueUpdateDraw(func() {
			agentsData = agentsList
			agentBox.Clear()
			if len(agentsData) == 0 {
				agentBox.AddItem("[gray]Scanning for agents...[-]", "", 0, nil)
			} else {
				for i, a := range agentsData {
					agentBox.AddItem(fmt.Sprintf(" [%d] %s", i+1, a.ID), fmt.Sprintf(" %s", a.Host), 0, nil)
				}
				selectedAgentIdx = nextSelectedIdx
				agentBox.SetCurrentItem(nextSelectedIdx)
			}

			if len(feedText) > 0 {
				feed.SetText(feedText)
			}

			if len(agentsData) == 0 {
				statusTable.SetTitle(" SYSTEM HEALTH (ACTIVE) ").SetTitleColor(tcell.ColorGreen)
				setTableRow(0, " CPU USAGE  ", "--", tcell.ColorGray)
				setTableRow(1, " NET RECV   ", "--", tcell.ColorGray)
				setTableRow(2, " NET SEND   ", "--", tcell.ColorGray)
				setTableRow(3, " STATUS     ", "SCANNING", tcell.ColorYellow)
				graphView.SetTitle(" METRIC GRAPHS ")
				graphView.SetText("[gray]Waiting for discovered agents...[-]")
				return
			}

			statusTable.SetTitle(statusTitle).SetTitleColor(statusColor)
			if viewMode == "live" {
				if metrics != nil {
					setTableRow(0, " CPU USAGE  ", fmt.Sprintf("%d%%", int(metrics.ComputeMetric.CPUUtilization)), tcell.ColorWhite)
					setTableRow(1, " NET RECV   ", fmt.Sprintf("%.2f KB/s", metrics.ComputeMetric.NetworkInBytes/1024), tcell.ColorWhite)
					setTableRow(2, " NET SEND   ", fmt.Sprintf("%.2f KB/s", metrics.ComputeMetric.NetworkOutBytes/1024), tcell.ColorWhite)
					setTableRow(3, " STATUS     ", "ONLINE", tcell.ColorGreen)
				} else {
					setTableRow(0, " CPU USAGE  ", "--", tcell.ColorGray)
					setTableRow(1, " NET RECV   ", "--", tcell.ColorGray)
					setTableRow(2, " NET SEND   ", "--", tcell.ColorGray)
					setTableRow(3, " STATUS     ", "NO DATA", tcell.ColorYellow)
				}
			} else if aggregate != nil {
				setTableRow(0, " AVG CPU      ", fmt.Sprintf("%.1f%%", aggregate.Average.CPUUtilization), tcell.ColorYellow)
				setTableRow(1, " PEAK NET I/O ", fmt.Sprintf("%.2f KB/s", aggregate.PeakNetworkBps/1024), tcell.ColorYellow)
				setTableRow(2, " SAMPLES      ", fmt.Sprintf("%d points", aggregate.SampleCount), tcell.ColorGray)
				windowLabel := "SINCE START"
				if aggregateWindow == "10m" {
					windowLabel = "LAST 10 MIN"
				}
				setTableRow(3, " WINDOW       ", windowLabel, tcell.ColorGreen)
			} else {
				setTableRow(0, " AVG CPU      ", "--", tcell.ColorGray)
				setTableRow(1, " PEAK NET I/O ", "--", tcell.ColorGray)
				setTableRow(2, " SAMPLES      ", "--", tcell.ColorGray)
				setTableRow(3, " WINDOW       ", strings.ToUpper(aggregateWindow), tcell.ColorGray)
			}

			updateGraphs(history, graphTitle)
		})
	}

	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			updateData()
		}
	}()
	go updateData()

	// Cleanup will be handled by main

	// Layout
	metricsFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(statusTable, 30, 0, false).
		AddItem(feed, 0, 1, false)

	monitorFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(metricsFlex, 10, 0, false).
		AddItem(graphView, 10, 0, false).
		AddItem(agentBox, 0, 1, true)

	// Add key handler for live/normal and aggregate window
	monitorFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'l' {
			viewMode = "live"
			feed.Write([]byte("[yellow]>> Switched to LIVE telemetry feed[-] \n"))
			go updateData()
		} else if event.Rune() == 'n' {
			viewMode = "normal"
			aggregateWindow = "overall"
			feed.Write([]byte("[yellow]>> Switched to NORMALIZED health view (overall aggregate)[-] \n"))
			go updateData()
		} else if event.Rune() == 'w' && viewMode == "normal" {
			if aggregateWindow == "overall" {
				aggregateWindow = "10m"
			} else {
				aggregateWindow = "overall"
			}
			feed.Write([]byte(fmt.Sprintf("[yellow]>> Normal mode aggregate window set to %s[-] \n", strings.ToUpper(aggregateWindow))))
			go updateData()
		}
		return event
	})

	return monitorFlex
}
