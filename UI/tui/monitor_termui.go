package tui

import (
	"fmt"
	"strings"
	"time"

	"cli/api"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func RunTermUIMonitor() error {
	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()

	viewMode := "live"
	aggregateWindow := "overall"
	selectedAgent := 0
	var agentsData []api.Agent
	var feedLines []string

	pushFeed := func(line string) {
		feedLines = append(feedLines, line)
		if len(feedLines) > 10 {
			feedLines = feedLines[len(feedLines)-10:]
		}
	}

	densifySeries := func(values []float64, target int) []float64 {
		if len(values) == 0 {
			return []float64{0, 0}
		}
		if len(values) == 1 {
			return []float64{values[0], values[0]}
		}
		if target <= len(values) {
			return values
		}

		out := make([]float64, target)
		last := float64(len(values) - 1)
		for i := 0; i < target; i++ {
			pos := (float64(i) / float64(target-1)) * last
			left := int(pos)
			right := left + 1
			if right >= len(values) {
				right = len(values) - 1
			}
			frac := pos - float64(left)
			out[i] = values[left] + (values[right]-values[left])*frac
		}
		return out
	}

	sparklineSeries := func(values []float64) ([]float64, float64) {
		values = densifySeries(values, 48)
		if len(values) == 0 {
			return []float64{0}, 1
		}

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

		rng := maxVal - minVal
		if rng <= 0 {
			return []float64{0, 0}, 1
		}

		out := make([]float64, len(values))
		padding := rng * 0.15
		for i, v := range values {
			out[i] = (v - minVal) + padding
		}
		return out, rng + padding
	}

	safeSeries := func(values []float64) []float64 {
		switch len(values) {
		case 0:
			return []float64{0, 0}
		case 1:
			return []float64{values[0], values[0]}
		default:
			return values
		}
	}

	maxOrDefault := func(values []float64, fallback float64) float64 {
		maxVal := 0.0
		for _, v := range values {
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal <= 0 {
			return fallback
		}
		return maxVal
	}

	safePieData := func(values []float64) []float64 {
		total := 0.0
		for _, v := range values {
			total += v
		}
		if total <= 0 {
			return []float64{1}
		}
		return values
	}

	header := widgets.NewParagraph()
	header.Title = "Chaos Monitor"
	header.Text = "[ q ] quit  [ up/down ] agents  [ l ] live  [ n ] normal  [ w ] overall/10m"
	header.BorderStyle.Fg = ui.ColorCyan
	header.TitleStyle.Fg = ui.ColorGreen
	header.TextStyle.Fg = ui.ColorWhite

	metrics := widgets.NewTable()
	metrics.Title = "System Health"
	metrics.RowSeparator = false
	metrics.TextStyle = ui.NewStyle(ui.ColorWhite)
	metrics.RowStyles[0] = ui.NewStyle(ui.ColorGreen, ui.ColorClear, ui.ModifierBold)
	metrics.BorderStyle.Fg = ui.ColorCyan
	metrics.TitleStyle.Fg = ui.ColorGreen

	feed := widgets.NewParagraph()
	feed.Title = "Live Telemetry Stream"
	feed.WrapText = false
	feed.BorderStyle.Fg = ui.ColorCyan
	feed.TitleStyle.Fg = ui.ColorGreen

	agentList := widgets.NewList()
	agentList.Title = "Discovered Host Agents"
	agentList.WrapText = false
	agentList.BorderStyle.Fg = ui.ColorWhite
	agentList.TitleStyle.Fg = ui.ColorGreen
	agentList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorCyan, ui.ModifierBold)

	sCPU := widgets.NewSparkline()
	sCPU.Title = "cpu"
	sCPU.LineColor = ui.ColorCyan
	sNet := widgets.NewSparkline()
	sNet.Title = "net"
	sNet.LineColor = ui.ColorRed
	sDisk := widgets.NewSparkline()
	sDisk.Title = "disk"
	sDisk.LineColor = ui.ColorYellow

	sparks := widgets.NewSparklineGroup(sCPU, sNet, sDisk)
	sparks.Title = "Recent Trends"
	sparks.BorderStyle.Fg = ui.ColorWhite
	sparks.TitleStyle.Fg = ui.ColorGreen

	bar := widgets.NewBarChart()
	bar.Title = "Current Metric Levels"
	bar.BarWidth = 6
	bar.BarGap = 2
	bar.Labels = []string{"CPU", "IN", "OUT", "DSK"}
	bar.BarColors = []ui.Color{ui.ColorGreen, ui.ColorCyan, ui.ColorBlue, ui.ColorYellow}
	bar.BorderStyle.Fg = ui.ColorWhite
	bar.TitleStyle.Fg = ui.ColorGreen
	bar.NumFormatter = func(v float64) string {
		if v > 0 && v < 10 {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%.0f", v)
	}
	bar.NumStyles = []ui.Style{ui.NewStyle(ui.ColorWhite, ui.ColorClear, ui.ModifierBold)}

	stacked := widgets.NewStackedBarChart()
	stacked.Title = "Selected vs Fleet Average"
	stacked.BarWidth = 7
	stacked.BarGap = 3
	stacked.Labels = []string{"CPU", "NET", "DSK"}
	stacked.BarColors = []ui.Color{ui.ColorCyan, ui.ColorMagenta}
	stacked.BorderStyle.Fg = ui.ColorWhite
	stacked.TitleStyle.Fg = ui.ColorGreen
	stacked.NumFormatter = func(v float64) string {
		if v > 0 && v < 10 {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%.0f", v)
	}
	stacked.NumStyles = []ui.Style{ui.NewStyle(ui.ColorWhite, ui.ColorClear, ui.ModifierBold)}

	pie := widgets.NewPieChart()
	pie.Title = "Current Resource Mix"
	pie.BorderStyle.Fg = ui.ColorWhite
	pie.TitleStyle.Fg = ui.ColorGreen
	pie.Colors = []ui.Color{ui.ColorRed, ui.ColorCyan, ui.ColorBlue, ui.ColorYellow}
	pie.LabelFormatter = func(dataIndex int, currentValue float64) string {
		labels := []string{"CPU", "IN", "OUT", "DSK"}
		if dataIndex >= 0 && dataIndex < len(labels) {
			return labels[dataIndex]
		}
		return ""
	}

	netPlot := widgets.NewPlot()
	netPlot.Title = "Network Receive Trend (KB/s)"
	netPlot.Marker = widgets.MarkerBraille
	netPlot.Data = make([][]float64, 1)
	netPlot.LineColors[0] = ui.ColorYellow
	netPlot.AxesColor = ui.ColorWhite
	netPlot.BorderStyle.Fg = ui.ColorWhite
	netPlot.TitleStyle.Fg = ui.ColorGreen

	fleetOverview := widgets.NewParagraph()
	fleetOverview.Title = "Fleet Overview"
	fleetOverview.WrapText = true
	fleetOverview.BorderStyle.Fg = ui.ColorWhite
	fleetOverview.TitleStyle.Fg = ui.ColorGreen
	fleetOverview.TextStyle.Fg = ui.ColorWhite

	trendInfo := widgets.NewParagraph()
	trendInfo.Title = "Panel Legend"
	trendInfo.WrapText = true
	trendInfo.BorderStyle.Fg = ui.ColorWhite
	trendInfo.TitleStyle.Fg = ui.ColorGreen
	trendInfo.TextStyle.Fg = ui.ColorWhite

	grid := ui.NewGrid()
	setGrid := func() {
		w, h := ui.TerminalDimensions()
		grid.SetRect(0, 0, w, h)
		grid.Set(
			ui.NewRow(0.12, header),
			ui.NewRow(0.24,
				ui.NewCol(0.27, metrics),
				ui.NewCol(0.46, feed),
				ui.NewCol(0.27, bar),
			),
			ui.NewRow(0.12,
				ui.NewCol(0.63, sparks),
				ui.NewCol(0.37, stacked),
			),
			ui.NewRow(0.22,
				ui.NewCol(0.38, pie),
				ui.NewCol(0.40, netPlot),
				ui.NewCol(0.22, fleetOverview),
			),
			ui.NewRow(0.08, trendInfo),
			ui.NewRow(0.22, agentList),
		)
	}
	setGrid()

	updateUI := func() {
		// Merge agents from both sources: management gateway + InfluxDB autodiscovery.
		// This ensures EC2 instances pushing metrics are always visible, even if
		// they are not registered with the gateway (e.g. new instances).
		seenIDs := make(map[string]bool)
		var agentsList []api.Agent

		if gwAgents, err := api.GetAgents(); err == nil {
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
		agentsData = agentsList
		if selectedAgent >= len(agentsData) {
			selectedAgent = max(0, len(agentsData)-1)
		}

		agentRows := make([]string, 0, len(agentsData))
		for i, a := range agentsData {
			agentRows = append(agentRows, fmt.Sprintf("[%d] %s  [%s]", i+1, a.ID, a.Host))
		}
		if len(agentRows) == 0 {
			agentRows = []string{"Scanning for agents..."}
		}
		agentList.Rows = agentRows
		agentList.SelectedRow = selectedAgent

		modeLabel := "LIVE"
		windowLabel := "LAST 1M"

		if len(agentsData) == 0 {
			metrics.Rows = [][]string{
				{"Metric", "Value"},
				{"CPU", "N/A"},
				{"NET", "N/A"},
				{"STATUS", "NO AGENTS"},
			}
			feed.Text = strings.Join(feedLines, "\n")
			sCPU.Data = []float64{0}
			sNet.Data = []float64{0}
			sDisk.Data = []float64{0}
			sCPU.MaxVal = 100
			sNet.MaxVal = 1
			sDisk.MaxVal = 1
			bar.Data = []float64{0, 0, 0, 0}
			bar.MaxVal = 1
			pie.Data = []float64{1}
			stacked.Data = [][]float64{{0, 0}, {0, 0}, {0, 0}}
			stacked.MaxVal = 1
			netPlot.Data[0] = []float64{0, 0}
			netPlot.MaxVal = 1
			fleetOverview.Text = "Agents online: 0\nAvg CPU: N/A\nAvg Net: N/A\nAvg Disk: N/A"
			trendInfo.Text = "Current Metric Levels = selected instance right now | Recent Trends = selected instance over current window | Pie = selected instance resource mix | Network Plot = selected instance inbound traffic trend | Selected vs Fleet = selected instance against fleet average."
			header.Text = "[ q ] quit  [ up/down ] agents  [ l ] live  [ n ] normal  [ w ] overall/10m"
			ui.Render(grid)
			return
		}

		targetAgent := agentsData[selectedAgent].ID
		latest, _ := api.GetComputeMetrics(targetAgent)
		history, _ := api.GetComputeHistory(targetAgent, 1*time.Minute)

		fleet, _ := api.GetFleetAggregate()
		fleetOnline := 0
		fleetAvgCPU := 0.0
		fleetAvgNet := 0.0
		fleetAvgDisk := 0.0

		if fleet != nil {
			fleetOnline = fleet.ActiveAgents // Simplified for now
			fleetAvgCPU = fleet.AverageCPU
			fleetAvgNet = fleet.AverageNet
			fleetAvgDisk = fleet.AverageDisk
		}

		if viewMode == "normal" {
			modeLabel = "NORMAL"
			if aggregateWindow == "10m" {
				windowLabel = "LAST 10M"
			} else {
				windowLabel = "OVERALL"
			}
			history, _ = api.GetComputeHistoryForScope(targetAgent, aggregateWindow)
			agg, _ := api.GetComputeAggregate(targetAgent, aggregateWindow)
			if agg != nil {
				latest = &api.ComputeSummary{
					InstanceID: targetAgent,
					ComputeMetric: api.ComputeMetric{
						CPUUtilization:  agg.Average.CPUUtilization,
						NetworkInBytes:  agg.Average.NetworkInBytes,
						NetworkOutBytes: agg.Average.NetworkOutBytes,
						DiskReadBytes:   agg.Average.DiskReadBytes,
						DiskWriteBytes:  agg.Average.DiskWriteBytes,
						StatusFailed:    agg.Average.StatusFailed,
					},
				}
			}
		}

		var cpuSeries []float64
		var netSeries []float64
		var diskSeries []float64
		var netInSeries []float64
		for _, point := range history {
			cpuSeries = append(cpuSeries, point.ComputeMetric.CPUUtilization)
			netSeries = append(netSeries, (point.ComputeMetric.NetworkInBytes+point.ComputeMetric.NetworkOutBytes)/1024)
			netInSeries = append(netInSeries, point.ComputeMetric.NetworkInBytes/1024)
			diskSeries = append(diskSeries, (point.ComputeMetric.DiskReadBytes+point.ComputeMetric.DiskWriteBytes)/1024)
		}

		if latest == nil {
			latest = &api.ComputeSummary{InstanceID: targetAgent}
		}

		cpuTrend, cpuTrendMax := sparklineSeries(cpuSeries)
		netTrend, netTrendMax := sparklineSeries(netSeries)
		diskTrend, diskTrendMax := sparklineSeries(diskSeries)
		netInSeries = densifySeries(netInSeries, 48)
		cpuSeries = densifySeries(cpuSeries, 48)
		netSeries = densifySeries(netSeries, 48)
		diskSeries = densifySeries(diskSeries, 48)

		status := "ONLINE"
		if latest.ComputeMetric.StatusFailed {
			status = "DEGRADED"
		}

		metrics.Title = fmt.Sprintf("System Health (%s)", modeLabel)
		metrics.Rows = [][]string{
			{"Metric", "Value"},
			{"CPU USAGE", fmt.Sprintf("%.1f%%", latest.ComputeMetric.CPUUtilization)},
			{"NET RECV", fmt.Sprintf("%.2f KB/s", latest.ComputeMetric.NetworkInBytes/1024)},
			{"NET SEND", fmt.Sprintf("%.2f KB/s", latest.ComputeMetric.NetworkOutBytes/1024)},
			{"DISK I/O", fmt.Sprintf("%.2f KB/s", (latest.ComputeMetric.DiskReadBytes+latest.ComputeMetric.DiskWriteBytes)/1024)},
			{"WINDOW", windowLabel},
			{"STATUS", status},
		}
		metrics.ColumnWidths = []int{12, 18}

		ts := time.Now().Format("15:04:05")
		pushFeed(fmt.Sprintf("%s Metric Packet | CPU: %.1f%% | Net: %.1fKb | %s", ts, latest.ComputeMetric.CPUUtilization, (latest.ComputeMetric.NetworkInBytes+latest.ComputeMetric.NetworkOutBytes)/1024, status))
		feed.Text = strings.Join(feedLines, "\n")

		sparks.Title = fmt.Sprintf("Recent Trends (%s)", windowLabel)
		sCPU.Title = "cpu %"
		sNet.Title = "net kb/s"
		sDisk.Title = "disk kb/s"
		sCPU.Data = cpuTrend
		sNet.Data = netTrend
		sDisk.Data = diskTrend
		sCPU.MaxVal = cpuTrendMax
		sNet.MaxVal = netTrendMax
		sDisk.MaxVal = diskTrendMax

		currentLevels := []float64{
			latest.ComputeMetric.CPUUtilization,
			latest.ComputeMetric.NetworkInBytes / 1024,
			latest.ComputeMetric.NetworkOutBytes / 1024,
			(latest.ComputeMetric.DiskReadBytes + latest.ComputeMetric.DiskWriteBytes) / 1024,
		}
		bar.Data = currentLevels
		bar.MaxVal = maxOrDefault(bar.Data, 1)
		pie.Data = safePieData(currentLevels)

		if fleet != nil {
			stacked.Data = [][]float64{
				{latest.ComputeMetric.CPUUtilization, fleetAvgCPU},
				{(latest.ComputeMetric.NetworkInBytes + latest.ComputeMetric.NetworkOutBytes) / 1024, fleetAvgNet},
				{(latest.ComputeMetric.DiskReadBytes + latest.ComputeMetric.DiskWriteBytes) / 1024, fleetAvgDisk},
			}
			stacked.MaxVal = maxOrDefault([]float64{
				stacked.Data[0][0] + stacked.Data[0][1],
				stacked.Data[1][0] + stacked.Data[1][1],
				stacked.Data[2][0] + stacked.Data[2][1],
			}, 1)
		}

		netPlot.Data[0] = safeSeries(netInSeries)
		netPlot.MaxVal = maxOrDefault(netPlot.Data[0], 1)
		netPlot.HorizontalScale = 1
		fleetOverview.Text = fmt.Sprintf(
			"Agents seen: %d\nHealthy now: %d\nAvg CPU: %.1f%%\nAvg Net: %.1f KB/s\nAvg Disk: %.1f KB/s",
			len(agentsData), fleetOnline, fleetAvgCPU, fleetAvgNet, fleetAvgDisk,
		)
		trendInfo.Text = fmt.Sprintf(
			"Selected now: CPU %.1f%% | Net In %.1f KB/s | Net Out %.1f KB/s | Disk %.1f KB/s. %s. Stacked bars compare selected instance against fleet average for CPU, total network, and disk I/O.",
			currentLevels[0], currentLevels[1], currentLevels[2], currentLevels[3], windowLabel,
		)

		header.Text = fmt.Sprintf("[ q ] quit  [ up/down ] agents  [ l ] live  [ n ] normal  [ w ] overall/10m    agent: %s    mode: %s %s",
			targetAgent, modeLabel, windowLabel)

		ui.Render(grid)
	}

	updateUI()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	uiEvents := ui.PollEvents()

	for {
		select {
		case <-ticker.C:
			updateUI()
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>", "<Escape>":
				return nil
			case "<Resize>":
				setGrid()
				ui.Clear()
				updateUI()
			case "<Down>":
				if selectedAgent < len(agentsData)-1 {
					selectedAgent++
					updateUI()
				}
			case "<Up>":
				if selectedAgent > 0 {
					selectedAgent--
					updateUI()
				}
			case "l":
				viewMode = "live"
				updateUI()
			case "n":
				viewMode = "normal"
				aggregateWindow = "overall"
				updateUI()
			case "w":
				if viewMode == "normal" {
					if aggregateWindow == "overall" {
						aggregateWindow = "10m"
					} else {
						aggregateWindow = "overall"
					}
					updateUI()
				}
			}
		}
	}
}

func clampPercent(v float64) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return int(v + 0.5)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
