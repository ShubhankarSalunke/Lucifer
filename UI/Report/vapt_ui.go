package report

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"github.com/adigajjar/security-audit/rules"
	glamour "github.com/charmbracelet/glamour"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

type Vulnerability struct {
	Title         string
	Severity      string
	Status        string
	SeverityValue int
	Content       string
}

func getSeverityBg(sev string) string {
	switch sev {
	case "CRITICAL":
		return "#ff4d4d"
	case "HIGH":
		return "#ffb84d"
	case "MEDIUM":
		return "#4da6ff"
	case "LOW":
		return "#33cc99"
	default:
		return "#444444"
	}
}

func getSeverityColor(sev string) string {
	switch sev {
	case "CRITICAL":
		return "red"
	case "HIGH":
		return "yellow"
	case "MEDIUM":
		return "blue"
	case "LOW":
		return "green"
	default:
		return "white"
	}
}

var found bool = true

func resolvePath(path string) string {
	if _, err := os.Stat(path); err == nil { return path }
	if _, err := os.Stat("../" + path); err == nil { return "../" + path }
	if _, err := os.Stat("../../" + path); err == nil { return "../../" + path }
	return path
}

func GetReport() []byte {
	path := resolvePath("security-audit/vapt_report_aws.md")
	if data, err := os.ReadFile(path); err == nil {
		found = true
		return data
	}
	found = false
	return nil
}

func saveRule(rule rules.Rule) error {
	path := "custom_rules.yaml"

	var custom rules.Rules
	data, _ := os.ReadFile(path)
	_ = yaml.Unmarshal(data, &custom)

	existingPath := resolvePath("security-audit/rules/rules.yaml")
	var existing rules.Rules
	data2, _ := os.ReadFile(existingPath)
	_ = yaml.Unmarshal(data2, &existing)

	for _, r := range existing.Rules {
		if r.ID == rule.ID {
			return fmt.Errorf("rule already exists")
		}
	}
	for _, r := range custom.Rules {
		if r.ID == rule.ID {
			return fmt.Errorf("custom rule already exists")
		}
	}

	custom.Rules = append(custom.Rules, rule)
	out, _ := yaml.Marshal(custom)
	return os.WriteFile(path, out, 0644)
}

func parseValue(v string) interface{} {
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return v
}

func parseMarkdown() []Vulnerability {
	content := string(GetReport())
	if idx := strings.Index(content, "## Detailed Findings"); idx != -1 {
		content = content[idx:]
	}
	blocks := strings.Split(content, "---")

	var vulns []Vulnerability

	titleRegex := regexp.MustCompile(`### \d+\. (.*?)\n`)
	severityRegex := regexp.MustCompile(`- \*\*Severity:\*\* (.*?)\n`)
	statusRegex := regexp.MustCompile(`- \*\*Status:\*\* (.*?)\n`)

	for _, block := range blocks {
		if !strings.Contains(block, "### ") {
			continue
		}

		v := Vulnerability{
			Content: strings.TrimSpace(block),
		}

		if m := titleRegex.FindStringSubmatch(block); len(m) > 1 {
			v.Title = m[1]
		}
		if m := severityRegex.FindStringSubmatch(block); len(m) > 1 {
			v.Severity = strings.TrimSpace(m[1])
		}
		if m := statusRegex.FindStringSubmatch(block); len(m) > 1 {
			v.Status = strings.TrimSpace(m[1])
		}

		switch v.Severity {
		case "CRITICAL":
			v.SeverityValue = 4
		case "HIGH":
			v.SeverityValue = 3
		case "MEDIUM":
			v.SeverityValue = 2
		case "LOW":
			v.SeverityValue = 1
		default:
			v.SeverityValue = 0
		}

		if v.Title != "" {
			vulns = append(vulns, v)
		}
	}

	sort.Slice(vulns, func(i, j int) bool {
		return vulns[i].SeverityValue > vulns[j].SeverityValue
	})

	return vulns
}

func extractField(content, field string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?i)%s:\s*(.*)`, field))
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		val := strings.TrimSpace(match[1])
		val = strings.ReplaceAll(val, "**", "")
		val = strings.ReplaceAll(val, "__", "")
		val = strings.ReplaceAll(val, "`", "")
		return val
	}
	return "N/A"
}

func trimRendered(s string) string {
	lines := strings.Split(s, "\n")
	max := 20
	if len(lines) > max {
		lines = lines[:max]
		lines = append(lines, "... (truncated)")
	}
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

func isTyping(p tview.Primitive) bool {
	switch v := p.(type) {
	case *tview.InputField:
		return v.HasFocus()
	case *tview.Form:
		return true
	default:
		return false
	}
}

func GetVaptReportView() tview.Primitive {
	reportData := GetReport()
	if reportData == nil || !found {
		return tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter).
			SetText("\n\n[gray]VAPT Report Not Found[-]\n[white]Please run a security scan first:[-]\n[cyan]security-audit aws all[-]").
			SetBorder(true).
			SetTitle(" VAPT REPORT ")
	}
	vulns := parseMarkdown()

	var filteredVulns []Vulnerability

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	filterFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

	// SEARCH
	searchInput := tview.NewInputField().
		SetLabel("  Search: ").
		SetFieldBackgroundColor(tcell.NewRGBColor(20, 25, 40)).
		SetFieldTextColor(tcell.ColorWhite)
	searchInput.SetBorder(true).SetTitle(" >_ SEARCH ")

	// FILTERS
	severityFilter := tview.NewDropDown().
		SetOptions([]string{"ALL", "CRITICAL", "HIGH", "MEDIUM", "LOW"}, nil)
	severityFilter.SetCurrentOption(0)
	severityFilter.SetBorder(true).SetTitle(" FILTER: SEVERITY ")

	statusFilter := tview.NewDropDown().
		SetOptions([]string{"ALL", "FAIL", "PASS"}, nil)
	statusFilter.SetCurrentOption(0)
	statusFilter.SetBorder(true).SetTitle(" FILTER: STATUS ")

	filterFlex.AddItem(searchInput, 0, 3, true)
	filterFlex.AddItem(severityFilter, 0, 1, false)
	filterFlex.AddItem(statusFilter, 0, 1, false)

	// LIST
	list := tview.NewList().ShowSecondaryText(true)
	list.SetBorder(true).SetTitle(" IDENTIFIED THREATS ")
	list.SetBackgroundColor(tcell.NewRGBColor(18, 22, 35))
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSecondaryTextColor(tcell.ColorGray)

	// DETAIL
	detailView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)

	detailView.SetBorder(true).
		SetTitle(" DETAILS ")

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("Report/my-theme.json"),
		glamour.WithWordWrap(0),
	)
	downloadBtn := tview.NewButton("Download").SetSelectedFunc(func() {
		data := GetReport()
		dst := filepath.Join(".", "vapt_report.md")
		err := os.WriteFile(dst, data, 0644)
		if err != nil {
			detailView.SetText("Error in Writing file")
			return
		}
		detailView.SetText("Report saved successfully")
	})
	rightPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(detailView, 0, 1, false).
		AddItem(downloadBtn, 1, 0, false)

	form := tview.NewForm()
	var rule rules.Rule
	form.SetFieldBackgroundColor(tcell.NewRGBColor(40, 45, 70))
	form.SetFieldTextColor(tcell.ColorWhite)
	form.AddInputField("ID", "", 20, nil, func(t string) { rule.ID = t })
	form.AddInputField("Name", "", 40, nil, func(t string) { rule.Name = t })
	form.SetButtonBackgroundColor(tcell.ColorDarkCyan)
	form.SetButtonTextColor(tcell.ColorWhite)
	form.AddDropDown("Severity",
		[]string{"CRITICAL", "HIGH", "MEDIUM", "LOW"},
		0,
		func(opt string, _ int) { rule.Severity = opt  },
		
	)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		}
		return event
	})
	form.AddInputField("Description", "", 60, nil, func(t string) { rule.Description = t })
	form.AddInputField("Type", "", 30, nil, func(t string) { rule.Type = t })
	form.AddInputField("Operator", "equals", 20, nil, func(t string) { rule.Check.Operator = t })
	form.AddInputField("Value", "", 20, nil, func(t string) { rule.Check.Value = parseValue(t) })
	form.AddInputField("Remediation", "", 60, nil, func(t string) { rule.Remediation = t })
	form.SetBorder(true).SetTitle(" ADD CUSTOM VAPT RULE ").SetBorderColor(tcell.ColorDarkCyan)
	form.AddButton("Save Rule", func() {
		_ = saveRule(rule)
	})

	populateList := func() {
		list.Clear()
		filteredVulns = []Vulnerability{}

		searchTerm := strings.ToLower(searchInput.GetText())
		_, sev := severityFilter.GetCurrentOption()
		_, stat := statusFilter.GetCurrentOption()

		for _, v := range vulns {
			if sev != "ALL" && v.Severity != sev {
				continue
			}
			if stat != "ALL" && v.Status != stat {
				continue
			}
			if searchTerm != "" &&
				!strings.Contains(strings.ToLower(v.Title), searchTerm) &&
				!strings.Contains(strings.ToLower(v.Content), searchTerm) {
				continue
			}

			filteredVulns = append(filteredVulns, v)

			statusText := "[green::b]PASS[-]"
			if strings.ToUpper(v.Status) == "FAIL" {
				statusText = "[red::b]FAIL[-]"
			}

			main := fmt.Sprintf("   [black:%s] %-58s [-:-:-]", getSeverityBg(v.Severity), v.Title)
			sec := fmt.Sprintf("  [%s]  [%s::b]%s[-]", statusText, getSeverityColor(v.Severity), v.Severity)

			list.AddItem(main, sec, 0, nil)
		}
	}

	updateDetail := func() {
		idx := list.GetCurrentItem()
		if idx < 0 || idx >= len(filteredVulns) {
			return
		}

		v := filteredVulns[idx]

		statusText := "[green::b]PASS[-]"
		if strings.ToUpper(v.Status) == "FAIL" {
			statusText = "[red::b]FAIL[-]"
		}

		var borderColor tcell.Color
		switch v.Severity {
		case "CRITICAL":
			borderColor = tcell.ColorRed
		case "HIGH":
			borderColor = tcell.ColorYellow
		case "MEDIUM":
			borderColor = tcell.ColorBlue
		case "LOW":
			borderColor = tcell.ColorGreen
		default:
			borderColor = tcell.ColorGray
		}

		detailView.SetBorderColor(borderColor)

		desc := extractField(v.Content, "Description")
		rem := extractField(v.Content, "Remediation")
		remediationText := fmt.Sprintf("\n[::b]Remediation[::-]\n[#cfcfcf]%s[-]\n", rem)
		if strings.ToUpper(v.Status) == "PASS" {
			remediationText = ""
		}

		rendered, err := renderer.Render(v.Content)
		if err != nil {
			rendered = v.Content
		}

		rendered = strings.ReplaceAll(rendered, "**", "")
		rendered = strings.ReplaceAll(rendered, "__", "")
		rendered = strings.ReplaceAll(rendered, "`", "")

		detailView.SetText(fmt.Sprintf(
			`[::b]%s[::-]

[gray]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

[white]Severity:[-] [%s::b]%s[-]     [white]Status:[-] %s

[gray]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

[::b]Description[::-]
[#cfcfcf]%s[-]

[gray]──────────────────────────────────────────────[-]
%s
[gray]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

[gray::i]Full Technical Details (sanitized)[-]

[#808080]%s[-]
`,
			v.Title,
			getSeverityColor(v.Severity),
			v.Severity,
			statusText,
			desc,
			remediationText,
			tview.TranslateANSI(trimRendered(rendered)),
		))

	}

	list.SetChangedFunc(func(int, string, string, rune) {
		updateDetail()
	})

	searchInput.SetChangedFunc(func(string) {
		populateList()
		list.SetCurrentItem(0)
		updateDetail()
	})

	severityFilter.SetSelectedFunc(func(string, int) {
		populateList()
		list.SetCurrentItem(0)
		updateDetail()
	})

	statusFilter.SetSelectedFunc(func(string, int) {
		populateList()
		list.SetCurrentItem(0)
		updateDetail()
	})

	if len(vulns) > 0 {
		populateList()
		updateDetail()
	}

	body := tview.NewFlex().
		AddItem(list, 0, 1, true).
		AddItem(rightPane, 0, 2, false)

	flex.AddItem(filterFlex, 5, 0, true)
	flex.AddItem(body, 0, 1, false)
	flex.AddItem(form, 8, 0, false)

	pages := tview.NewPages()
	pages.AddPage("main", flex, true, true)
	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		_, focus := pages.GetFrontPage()

		if isTyping(focus) || form.HasFocus() {
			return event
		}

		return event
	})

	return pages
}
