package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"math"

	"cli/api"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/rivo/tview"
	"github.com/ShubhankarSalunke/chaos-engineering/datamodel"
)


// MonitorState stores the history for the termui dashboard so it persists across tab switches
type MonitorState struct {
	CPUHistory  []float64
	MemHistory  []float64
	LatHistory  []float64



	FeedLines   []string
	SelectedIdx int
	ExpIdx              int
	LastExpSeen         map[string]bool
	LastSelectedAgentID string
}


var GlobalMonitorState = &MonitorState{
	CPUHistory:  make([]float64, 50),
	MemHistory:  make([]float64, 50),
	LatHistory:  make([]float64, 50),

	FeedLines:   make([]string, 0),
	LastExpSeen: make(map[string]bool),
}



func RunIntro() string {
	app := tview.NewApplication()
	tview.Styles.PrimitiveBackgroundColor = TrueBlack
	
	next := "Quit"
	intro := NewIntroScreen(app, func() {
		next = "Main"
		app.Stop()
	})

	if err := app.SetRoot(intro, true).Run(); err != nil {
		return "Quit"
	}
	return next
}


func RunTermUIMonitor() string {
	if err := ui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v\n", err)
		return "Main"
	}
	defer ui.Close()

	selectedAgent := GlobalMonitorState.SelectedIdx
	selectedExp := GlobalMonitorState.ExpIdx
	activeList := "agents"

	ensureHistory := func(s []float64) []float64 {
		if len(s) == 0 { return []float64{0, 0} }
		if len(s) == 1 { return []float64{s[0], s[0]} }
		return s
	}

	header := widgets.NewParagraph()
	header.Title = "Chaos Monitor"
	header.Text = "[ q ] back  [ 1 ] agents  [ 2 ] history  [ tab ] switch  [ j/k ] scroll"
	header.BorderStyle.Fg = ui.ColorCyan

	metricsTable := widgets.NewTable()
	metricsTable.Title = " System Health "
	metricsTable.RowSeparator = false
	metricsTable.BorderStyle.Fg = ui.ColorCyan
	metricsTable.Rows = [][]string{
		{"Metric", "Value", "Status"},
	}

	feed := widgets.NewParagraph()
	feed.Title = "Live Activity Feed (Pinned: Active)"
	feed.BorderStyle.Fg = ui.ColorCyan

	agentList := widgets.NewList()
	agentList.Title = " Agents (1) "
	agentList.BorderStyle.Fg = ui.ColorYellow
	agentList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorCyan)

	expList := widgets.NewList()
	expList.Title = " Experiment History (2) "
	expList.BorderStyle.Fg = ui.ColorWhite
	expList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorYellow)

	cpuPlot := widgets.NewPlot()
	cpuPlot.Title = "CPU Impact (%)"
	cpuPlot.Marker = widgets.MarkerBraille
	cpuPlot.Data = [][]float64{{0, 0}}
	cpuPlot.LineColors[0] = ui.ColorCyan




	timerGauge := widgets.NewGauge()
	timerGauge.Title = "Experiment Progress"
	timerGauge.Percent = 0
	timerGauge.BarColor = ui.ColorRed
	timerGauge.LabelStyle = ui.NewStyle(ui.ColorWhite)
	timerGauge.TitleStyle = ui.NewStyle(ui.ColorCyan)


	memPlot := widgets.NewPlot()
	memPlot.Title = "Memory Impact (MB)"
	memPlot.Marker = widgets.MarkerBraille
	memPlot.Data = [][]float64{{0, 0}}
	memPlot.LineColors[0] = ui.ColorMagenta

	latPlot := widgets.NewPlot()
	latPlot.Title = "Latency Impact (ms)"
	latPlot.Marker = widgets.MarkerBraille
	latPlot.Data = [][]float64{{0, 0}}
	latPlot.LineColors[0] = ui.ColorGreen




	vaptBar := widgets.NewBarChart()
	vaptBar.Title = " VAPT Findings (FAILs) "
	vaptBar.Data = []float64{0, 0, 0, 0, 0}
	vaptBar.Labels = []string{"EC2", "S3", "IAM", "RDS", "LAM"}
	vaptBar.BarWidth = 5
	vaptBar.BarColors = []ui.Color{ui.ColorRed}
	vaptBar.LabelStyles = []ui.Style{ui.NewStyle(ui.ColorCyan)}
	vaptBar.NumStyles = []ui.Style{ui.NewStyle(ui.ColorWhite)}

	vaptMetrics := widgets.NewParagraph()
	vaptMetrics.Title = " VAPT Security Metrics "
	vaptMetrics.BorderStyle.Fg = ui.ColorGreen

	fleetPara := widgets.NewParagraph()
	fleetPara.Title = " Fleet Metrics "
	fleetPara.BorderStyle.Fg = ui.ColorWhite

	taskDetail := widgets.NewParagraph()
	taskDetail.Title = " Detailed Task Mapping "
	taskDetail.BorderStyle.Fg = ui.ColorYellow
	taskDetail.Text = "Loading task details..."

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(0.05, header),
		ui.NewRow(0.95,
			ui.NewCol(0.2,
				ui.NewRow(0.3, agentList),
				ui.NewRow(0.7, expList),
			),

			ui.NewCol(0.8,
				ui.NewRow(0.2,
					ui.NewCol(0.4, metricsTable),
					ui.NewCol(0.6, feed),
				),
				ui.NewRow(0.25,
					ui.NewCol(0.85, cpuPlot),
					ui.NewCol(0.15, timerGauge),
				),


				ui.NewRow(0.25,
					ui.NewCol(0.5, memPlot),
					ui.NewCol(0.5, latPlot),
				),

				ui.NewRow(0.3,
					ui.NewCol(0.2, vaptMetrics),
					ui.NewCol(0.3, fleetPara),
					ui.NewCol(0.2, vaptBar),
					ui.NewCol(0.3, taskDetail),
				),

			),
		),
	)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()



	uiEvents := ui.PollEvents()

	var lastAgentCount int
	var lastExpCount int
	var lastVaptScore float64
	var lastStats datamodel.FleetStats
	var lastSelectedAgent int
	var lastSelectedExp int
	firstRun := true

	var (
		isExperimentRunning = false
		lastExperimentState = ""
		cloudWatchTicker    *time.Ticker
		resultPollTicker    *time.Ticker
	)

	
	cloudWatchTicker = time.NewTicker(30 * time.Second)
	resultPollTicker = time.NewTicker(3 * time.Second) // Check experiment status frequently
	
	defer cloudWatchTicker.Stop()
	defer resultPollTicker.Stop()

	update := func(useCloudWatch bool) {
		stats, err := datamodel.GetFleetStats()
		if err != nil { 
			stats = &datamodel.FleetStats{} 
		}

		activeCount, activeIDs := datamodel.GetActiveExperimentCount()
		
		results, _ := api.GetResults()
		
		// Convert raw results to ExperimentResult map
		typedResults := make(map[string]datamodel.ExperimentResult)
		for id, res := range results {
			res.ExperimentID = id
			typedResults[id] = res
		}

		
		// Determine experiment state
		hasRunningExperiments := false
		hasPendingExperiments := false
		
		// Check actual experiment status from API
		allExperiments, _ := api.GetExperiments()
		if allExperiments != nil {
			for _, exp := range allExperiments {
				status := exp.Status
				if status == "IN_PROGRESS" || status == "running" {
					hasRunningExperiments = true
				} else if status == "PENDING" || status == "pending" {
					hasPendingExperiments = true
				}
			}
		}
		
		// Update state and manage tickers
		newState := "IDLE"
		if hasRunningExperiments {
			newState = "RUNNING"
		} else if hasPendingExperiments {
			newState = "PENDING"
		}
		
		// State change detection
		if newState != lastExperimentState {
			lastExperimentState = newState
			
			if newState == "RUNNING" {
				cloudWatchTicker.Stop()
				cloudWatchTicker = time.NewTicker(24 * time.Hour)
				isExperimentRunning = true
			} else {
				// Resume CloudWatch
				cloudWatchTicker.Stop()
				cloudWatchTicker = time.NewTicker(30 * time.Second)
				isExperimentRunning = false
			}
		}

		if activeList == "agents" {

			agentList.BorderStyle.Fg = ui.ColorYellow
			expList.BorderStyle.Fg = ui.ColorWhite
		} else {
			agentList.BorderStyle.Fg = ui.ColorWhite
			expList.BorderStyle.Fg = ui.ColorYellow
		}

		fleetPara.Text = fmt.Sprintf("\n  [Active Agents: ](fg:green) %d\n  [Running Tasks: ](fg:yellow) %d\n  [Avg Latency:   ](fg:cyan) %.1f ms\n  [Avg Memory:    ](fg:magenta) %.1f MB\n  [Avg CPU Spike: ](fg:cyan) %.1f%%",
			stats.ActiveAgents, activeCount, stats.AverageLatency, stats.AverageMemory, stats.AverageCPU)


		agents, err := api.GetDiscoveredAgents()

		if err != nil {
			agents = []datamodel.Agent{}
		}

		if len(agents) != lastAgentCount || lastAgentCount == 0 {
			var agentNames []string
			for _, a := range agents { 
				// Only show active agents (heartbeats) or specifically named agents
				if a.Status == "Active" || strings.Contains(strings.ToLower(a.ID), "agent") {
					agentNames = append(agentNames, a.ID) 
				}
			}
			agentList.Rows = agentNames
			lastAgentCount = len(agents)
		}

		if selectedAgent >= len(agentList.Rows) { selectedAgent = len(agentList.Rows) - 1 }
		if selectedAgent < 0 { selectedAgent = 0 }
		agentList.SelectedRow = selectedAgent

		currentAgentID := ""
		if len(agentList.Rows) > 0 { currentAgentID = agentList.Rows[selectedAgent] }

		vaptFindings := datamodel.GetVAPTFindings()
		var failedRows []string
		var passedRows []string
		var failedMap = make(map[string]bool)

		for _, f := range vaptFindings {
			row := "audit_" + f.RuleID
			if strings.ToUpper(f.Status) == "FAIL" {
				failedRows = append(failedRows, row)
				failedMap[row] = true
			} else {
				passedRows = append(passedRows, row)
			}
		}

		var treeRows []string
		var displayRows []string

		if len(failedRows) > 0 {
			treeRows = append(treeRows, "HEADER_FAILED")
			displayRows = append(displayRows, "[▼ FAILED AUDITS](fg:red,mod:bold)")
			for _, r := range failedRows {
				treeRows = append(treeRows, r)
				displayRows = append(displayRows, "  "+r)
			}
		}
		
		activeChaosCount := 0
		for _, id := range datamodel.GetExperimentIDsByAgent(currentAgentID) {
			if !strings.HasPrefix(id, "audit_") {
				if activeChaosCount == 0 {
					treeRows = append(treeRows, "HEADER_CHAOS")
					displayRows = append(displayRows, "[▼ ACTIVE CHAOS](fg:cyan,mod:bold)")
				}
				treeRows = append(treeRows, id)
				displayRows = append(displayRows, "  "+id)
				activeChaosCount++
			}
		}

		if len(passedRows) > 0 {
			treeRows = append(treeRows, "HEADER_PASSED")
			displayRows = append(displayRows, "[▼ PASSED AUDITS](fg:green,mod:bold)")
			for _, r := range passedRows {
				treeRows = append(treeRows, r)
				displayRows = append(displayRows, "  "+r)
			}
		}

		expList.Rows = displayRows
		lastExpCount = len(displayRows)
		
		if selectedExp >= len(expList.Rows) { selectedExp = len(expList.Rows) - 1 }
		if selectedExp < 0 { selectedExp = 0 }
		expList.SelectedRow = selectedExp

		if len(treeRows) > 0 && selectedExp < len(treeRows) {
			currentExpID := treeRows[selectedExp]
			
			if strings.HasPrefix(currentExpID, "HEADER_") {
				taskDetail.Text = "\n\n  [Select a task to view forensics...](fg:gray)"
			} else {
				res, hasRes := typedResults[currentExpID]
				desc := datamodel.GetMappedDescription(currentExpID)

				if hasRes {
					recovery := "[PENDING](fg:yellow)"
					if res.Restored { recovery = "[SUCCESS](fg:green)" }
					
					impact := res.Impact
					if impact == "" { 
						impact = fmt.Sprintf("[CPU:](fg:cyan)%d%% [MEM:](fg:yellow)%dMB [LAT:](fg:magenta)%dms", 
							res.CPUPercent, res.MemoryMB, res.LatencyMS)
					}

					latestObs := "No observations yet."
					if len(res.Observations) > 0 {
						latestObs = res.Observations[len(res.Observations)-1].Message
					}
					
					diffInfo := ""
					if len(res.SnapshotDiff) > 0 {
						var b strings.Builder
						for k, v := range res.SnapshotDiff {
							fmt.Fprintf(&b, " %s:%v", k, v)
							if b.Len() > 120 { b.WriteString("..."); break }
						}
						diffInfo = "\n[MUTATIONS:](fg:magenta)" + b.String()
					}

					dispID := currentExpID
					if strings.HasPrefix(dispID, "audit_") {dispID = strings.TrimPrefix(dispID, "audit_") }
					if len(dispID) > 12 { dispID = dispID[:12] }

					if strings.HasPrefix(currentExpID, "audit_") && strings.ToUpper(res.Status) == "PASS" {
						if idx := strings.Index(desc, "[REMEDIATION:]"); idx != -1 {
							desc = desc[:idx]
						}
					}

					taskDetail.Text = fmt.Sprintf("[ID:  ](fg:cyan)%s  [STATUS:](fg:white)%s\n[DESC:](fg:yellow) %s\n[IMPACT:](fg:white) %s\n[RESTORE:](fg:white)%s  [LOG:](fg:cyan) %s%s", 
						dispID, res.Status, desc, impact, recovery, latestObs, diffInfo)

				} else {
					taskDetail.Text = fmt.Sprintf("[ID:  ](fg:cyan)%s\n[DESC:](fg:yellow) %s\n[INFO:](fg:white) Waiting for result telemetry...", 
						currentExpID, desc)
				}
			}
		}

		if stats.VaptScore != lastVaptScore || firstRun {
			findings := datamodel.GetVAPTFindings()
			total := len(findings)
			passed := 0
			failed := 0
			critical := 0
			high := 0
			
			counts := map[string]float64{"EC2": 0, "S3": 0, "IAM": 0, "RDS": 0, "LAM": 0}
			for _, f := range findings {
				if strings.ToUpper(f.Status) == "PASS" {
					passed++
				} else {
					failed++
					if strings.ToUpper(f.Severity) == "CRITICAL" { critical++ }
					if strings.ToUpper(f.Severity) == "HIGH" { high++ }
					
					parts := strings.Split(f.RuleID, "-")
					if len(parts) > 1 {
						if _, ok := counts[parts[1]]; ok { counts[parts[1]]++ }
					}
				}
			}

			vaptMetrics.Text = fmt.Sprintf("\n [Total Rules: ](fg:cyan) %d\n [Passed:      ](fg:green) %d\n [Failed:      ](fg:red) %d\n [Critical:    ](fg:red,mod:bold) %d\n [High Risks:  ](fg:yellow) %d",
				total, passed, failed, critical, high)
			
			if stats.VaptScore < 40 { vaptMetrics.BorderStyle.Fg = ui.ColorRed } else if stats.VaptScore < 75 { vaptMetrics.BorderStyle.Fg = ui.ColorYellow } else { vaptMetrics.BorderStyle.Fg = ui.ColorGreen }
			
			vaptBar.Data = []float64{counts["EC2"], counts["S3"], counts["IAM"], counts["RDS"], counts["LAM"]}
			lastVaptScore = stats.VaptScore
		
			currentAgentID := ""
			if len(agentList.Rows) > 0 && selectedAgent < len(agentList.Rows) {
				currentAgentID = strings.Trim(agentList.Rows[selectedAgent], "\"")
			}
			if currentAgentID != GlobalMonitorState.LastSelectedAgentID {
				GlobalMonitorState.CPUHistory = make([]float64, 50)
				GlobalMonitorState.MemHistory = make([]float64, 50)
				GlobalMonitorState.LatHistory = make([]float64, 50)

				GlobalMonitorState.LastSelectedAgentID = currentAgentID
				GlobalMonitorState.LastExpSeen = make(map[string]bool)
				
				for id, res := range typedResults {
					if res.TargetID == currentAgentID || strings.Contains(strings.ToLower(id), strings.ToLower(currentAgentID)) {
						if res.Status == "completed" || res.Status == "COMPLETED" {

							for i := 0; i < 3; i++ { 
								GlobalMonitorState.CPUHistory = append(GlobalMonitorState.CPUHistory[1:], float64(res.CPUPercent))
								GlobalMonitorState.MemHistory = append(GlobalMonitorState.MemHistory[1:], float64(res.MemoryMB))
								GlobalMonitorState.LatHistory = append(GlobalMonitorState.LatHistory[1:], float64(res.LatencyMS))
							}

							GlobalMonitorState.CPUHistory = append(GlobalMonitorState.CPUHistory[1:], 0)
							GlobalMonitorState.MemHistory = append(GlobalMonitorState.MemHistory[1:], 0)
							GlobalMonitorState.LatHistory = append(GlobalMonitorState.LatHistory[1:], 0)

							GlobalMonitorState.LastExpSeen[id] = true
						}
					}
				}	
			}
		}
		// 2. LIVE SCROLLING (Always run every tick)
		// Add a tiny sine wave for "wavy" look
		osc := 0.2 * math.Sin(float64(time.Now().Unix())/2.0)
		GlobalMonitorState.CPUHistory = append(GlobalMonitorState.CPUHistory[1:], 0.5 + osc)
		GlobalMonitorState.MemHistory = append(GlobalMonitorState.MemHistory[1:], 10.0 + (osc*5))
		GlobalMonitorState.LatHistory = append(GlobalMonitorState.LatHistory[1:], 1.0 + osc)

		// 3. NEW SPIKE INJECTION

		for id, res := range typedResults {
			if !GlobalMonitorState.LastExpSeen[id] && (res.Status == "completed" || res.Status == "COMPLETED") {
				if strings.ToLower(res.TargetID) == strings.ToLower(currentAgentID) || strings.Contains(strings.ToLower(id), strings.ToLower(currentAgentID)) {
					// Spike found for current agent!
					for i := 0; i < 5; i++ {
						if res.CPUPercent > 0 { GlobalMonitorState.CPUHistory = append(GlobalMonitorState.CPUHistory[1:], float64(res.CPUPercent)) }
						if res.MemoryMB > 0 { GlobalMonitorState.MemHistory = append(GlobalMonitorState.MemHistory[1:], float64(res.MemoryMB)) }
						if res.LatencyMS > 0 { GlobalMonitorState.LatHistory = append(GlobalMonitorState.LatHistory[1:], float64(res.LatencyMS)) }
					}
					GlobalMonitorState.LastExpSeen[id] = true
				}
			}
		}

		// 4. Timer Gauge Logic
		activeProgress := 0
		for _, res := range typedResults {
			if strings.ToLower(res.Status) == "running" || strings.ToLower(res.Status) == "in_progress" {

				startStr := strings.Split(res.CreatedAt, " m=")[0]
				start, err := time.Parse("2006-01-02 15:04:05.999999 -0700 MST", startStr)
				if err == nil {
					elapsed := time.Since(start).Seconds()
					if res.Duration > 0 {
						activeProgress = int((elapsed / float64(res.Duration)) * 100)
						if activeProgress > 100 { activeProgress = 100 }
						timerGauge.Title = fmt.Sprintf(" Experiment: %s (%ds remaining) ", res.ExperimentType, res.Duration-int(elapsed))
					}
				}
				break 
			}
		}
		if activeProgress == 0 {
			timerGauge.Title = " No Active Experiment "
		}
		timerGauge.Percent = activeProgress

		cpuPlot.Data = [][]float64{ensureHistory(GlobalMonitorState.CPUHistory)}
		memPlot.Data = [][]float64{ensureHistory(GlobalMonitorState.MemHistory)}
		latPlot.Data = [][]float64{ensureHistory(GlobalMonitorState.LatHistory)}


		var activeLines []string
		for _, id := range activeIDs {
			if strings.HasPrefix(id, "audit_") { continue }
			shortID := id
			if len(shortID) > 8 { shortID = shortID[:8] }
			activeLines = append(activeLines, fmt.Sprintf("[LIVE:  ](fg:yellow,mod:bold) [IN_PROGRESS](fg:white) %s", shortID))
		}

		var resultIDs []string
		for id := range typedResults {
			if !strings.HasPrefix(id, "audit_") { resultIDs = append(resultIDs, id) }
		}
		sort.Strings(resultIDs)

		var resultLines []string
		for _, id := range resultIDs {
			res := typedResults[id]
			shortID := id
			if len(shortID) > 8 { shortID = shortID[:8] }
			color := "green"
			if res.Status == "FAIL" { color = "red" }
			resultLines = append(resultLines, fmt.Sprintf("[%s:](fg:%s) [COMPLETED](fg:white) %s", shortID, color, res.Status))
		}
		
		allLines := append(activeLines, resultLines...)
		if len(allLines) > 10 { allLines = allLines[:10] }
		feed.Text = strings.Join(allLines, "\n")



		var toRender []ui.Drawable

		if stats.AverageCPU != lastStats.AverageCPU || firstRun {
			metricsTable.Rows = [][]string{
				{"Metric", "Value", "Status"},
				{"Fleet CPU", fmt.Sprintf("%.1f%%", stats.AverageCPU), "OK"},
				{"Fleet Mem", fmt.Sprintf("%.1f MB", stats.AverageMemory), "OK"},
				{"VAPT PASS", fmt.Sprintf("%.0f%%", stats.VaptScore), "HEALTHY"},
			}
			toRender = append(toRender, metricsTable)

			fleetPara.Text = fmt.Sprintf("\n  [Active Agents: ](fg:green) %d\n  [Running Tasks: ](fg:yellow) %d\n  [Avg Latency:   ](fg:cyan) %.1f ms\n  [Avg Memory:    ](fg:magenta) %.1f MB\n  [Avg CPU Spike: ](fg:cyan) %.1f%%",
				stats.ActiveAgents, activeCount, stats.AverageLatency, stats.AverageMemory, stats.AverageCPU)
			toRender = append(toRender, fleetPara)
		}


		if len(agents) != lastAgentCount || selectedAgent != lastSelectedAgent || firstRun {
			toRender = append(toRender, agentList)
			lastSelectedAgent = selectedAgent
		}
		if lastExpCount != len(displayRows) || selectedExp != lastSelectedExp || firstRun {
			toRender = append(toRender, expList)
			lastSelectedExp = selectedExp
		}

		toRender = append(toRender, feed)


		metrics := []ui.Drawable{cpuPlot, memPlot, latPlot, timerGauge}
		if metrics != nil || firstRun {
			toRender = append(toRender, cpuPlot, memPlot, latPlot, timerGauge)
		}


		if stats.VaptScore != lastVaptScore || firstRun {
			toRender = append(toRender, vaptBar, vaptMetrics)
		}
		
		toRender = append(toRender, taskDetail)

		if firstRun {
			ui.Render(grid)
			firstRun = false
		} else if len(toRender) > 0 {
			ui.Render(toRender...)
		}
		
		lastStats = *stats
	}

	update(true)

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				GlobalMonitorState.SelectedIdx = selectedAgent
				GlobalMonitorState.ExpIdx = selectedExp
				return "Main"
			case "1":
				activeList = "agents"
				update(false)
			case "2":
				activeList = "exps"
				update(false)
			case "<Down>":
				if activeList == "agents" && len(agentList.Rows) > 0 {
					if selectedAgent < len(agentList.Rows)-1 { selectedAgent++ }
				} else if activeList == "exps" && len(expList.Rows) > 0 {
					if selectedExp < len(expList.Rows)-1 { selectedExp++ }
				}
				update(false)
			case "<Up>":
				if activeList == "agents" {
					if selectedAgent > 0 { selectedAgent-- }
				} else {
					if selectedExp > 0 { selectedExp-- }
				}
				update(false)
			case "s":
				go func() { api.SyncAgents() }()
				update(false)
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			}
			
		case <-cloudWatchTicker.C:
			if !isExperimentRunning {
				update(true) 
			}
			
		case <-resultPollTicker.C:
			update(false) 
		}
	}
}
