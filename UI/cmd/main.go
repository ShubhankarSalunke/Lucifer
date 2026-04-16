package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	report "cli/Report"
	"cli/tui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println(`
		CHAOS CONTROL TERMINAL - Help

		Usage:
		go run cmd/main.go [--help]
		make ui-help

		Navigation:
		Press Esc        Return to tab navigation bar
		Press 1          Switch to Monitor Dashboard
		Press 2          Switch to Lucifer CLI
		Press 3          Switch to VAPT Security Report
		Press 4 or q     Exit
		`)
		os.Exit(0)
	}

	state := "Monitor"
	for {
		switch state {
		case "Monitor":
			if err := tui.RunTermUIMonitor(); err != nil {
				log.Printf("failed to run termui monitor: %v", err)
			}
			state = "Main"
		default:
			state = runCLI()
		}
		if state == "Quit" {
			break
		}
	}
}

func tabLabel(active string) string {
	tabs := []struct{ id, label string }{
		{"0", "MONITOR"},
		{"1", "LUCIFER"},
		{"2", "VAPT"},
		{"3", "EXIT"},
	}
	out := " "
	for _, t := range tabs {
		if t.id == active {
			out += fmt.Sprintf(`["%s"][white::rb]  %s  [""]  `, t.id, t.label)
		} else {
			out += fmt.Sprintf(`["%s"][gray::-]  %s  [""]  `, t.id, t.label)
		}
	}
	return out
}

func runCLI() string {
	tview.Styles.PrimitiveBackgroundColor = tcell.NewRGBColor(12, 12, 14)
	tview.Styles.ContrastBackgroundColor = tcell.NewRGBColor(20, 20, 24)
	tview.Styles.MoreContrastBackgroundColor = tcell.NewRGBColor(28, 28, 32)
	tview.Styles.BorderColor = tcell.NewRGBColor(60, 60, 70)
	tview.Styles.TitleColor = tcell.NewRGBColor(180, 180, 200)
	tview.Styles.PrimaryTextColor = tcell.NewRGBColor(220, 220, 230)
	tview.Styles.SecondaryTextColor = tcell.NewRGBColor(140, 140, 160)
	tview.Styles.TertiaryTextColor = tcell.NewRGBColor(90, 90, 110)

	app := tview.NewApplication()
	var nextState string = "Quit"

	notifyChan := make(chan string, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	header := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	header.SetText(`[white::b]CHAOS CONTROL TERMINAL[-::-]   [#3c3c46]| Esc = nav  | run 'make ui-help' for usage[-]`)

	tabBar := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetTextAlign(tview.AlignCenter)
	tabBar.SetBorder(true).SetTitle(" NAVIGATION ").SetTitleColor(tcell.NewRGBColor(160, 160, 180))
	tabBar.SetText(tabLabel("0"))

	notificationBar := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	notificationBar.SetBorder(true).SetTitle(" SYSTEM LOG ").SetTitleColor(tcell.NewRGBColor(160, 160, 180))
	notificationBar.SetText("[#3c8c5c]SYSTEM ONLINE[-]   [#3c3c46]Press Esc then a number to switch tabs[-]")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-notifyChan:
				app.QueueUpdateDraw(func() {
					notificationBar.SetText(fmt.Sprintf("[#3c3c46]%s[-]  %s", time.Now().Format("15:04:05"), msg))
				})
			}
		}
	}()

	pages := tview.NewPages()

	termOutput := tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	termOutput.SetBorder(true).SetTitle(" LUCIFER SESSION OUTPUT ").SetTitleColor(tcell.ColorGreen)
	fmt.Fprintf(termOutput, "[gray]Lucifer CLI Terminal ready. Type 'help', 'signup <email>', or 'login <token>' to begin.[-]\n\n")

	nextStepBox := tview.NewTextView().SetDynamicColors(true).
		SetWrap(true)
	nextStepBox.SetBorder(true).
		SetTitle(" NEXT BEST STEP ").
		SetTitleColor(tcell.ColorYellow)
	nextStepBox.SetText("[yellow]Start here:[-] `signup <email>` if you are new, or `login <token>` if you already have a Chaos API token.\n[gray]After auth:[-] try `create-agent --agent <instance-id>` or `create-experiment --type cpu_stress --agent <id> --duration 30 --cpu 40`.")

	termInput := tview.NewInputField().SetLabel("[green]> [-]").SetFieldWidth(0)
	termInput.SetBorder(true).SetTitle(" INTERACTIVE COMMANDS ").SetTitleColor(tcell.ColorGreen)

	resolveLuciferCLIDir := func() (string, error) {
		candidates := []string{
			filepath.Join("..", "..", "..", "Lucifer", "cli"),
			filepath.Join("..", "..", "Lucifer", "cli"),
			filepath.Join("Lucifer", "cli"),
		}

		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}

		for _, candidate := range candidates {
			dir := filepath.Clean(filepath.Join(cwd, candidate))
			if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
				return dir, nil
			}
		}

		return "", fmt.Errorf("could not locate Lucifer CLI directory from %s", cwd)
	}

	termInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			cmdText := termInput.GetText()
			if cmdText == "" {
				return
			}
			termInput.SetText("")

			fmt.Fprintf(termOutput, "[white::b]> %s[-] \n", cmdText)

			go func(c string) {
				args := strings.Fields(c)
				if len(args) == 0 {
					return
				}

				cliDir, err := resolveLuciferCLIDir()
				if err != nil {
					app.QueueUpdateDraw(func() {
						fmt.Fprintf(termOutput, "[red]CLI path resolution failed: %v[-]\n", err)
						termOutput.ScrollToEnd()
					})
					return
				}

				stdinPayload := []byte(nil)
				switch args[0] {
				case "login":
					app.QueueUpdateDraw(func() {
						nextStepBox.SetText("[yellow]Logging in:[-] paste your token after `login`.\n[gray]After a successful login:[-] try `create-agent --agent <instance-id>` to bind an agent, then `create-experiment ...`.")
					})
					if len(args) < 2 {
						app.QueueUpdateDraw(func() {
							fmt.Fprintf(termOutput, "[yellow]Usage in TUI: login <token>[-]\n")
							termOutput.ScrollToEnd()
						})
						return
					}
					stdinPayload = []byte(args[1] + "\n")
					args = []string{"login"}
				case "signup":
					app.QueueUpdateDraw(func() {
						nextStepBox.SetText("[yellow]Signing up:[-] use your email after `signup`.\n[gray]After signup finishes:[-] your token is saved, so the next step is usually `create-agent --agent <instance-id>`.")
					})
					userID := ""
					if len(args) > 1 {
						userID = strings.Join(args[1:], " ")
					}
					stdinPayload = []byte(userID + "\n")
					args = []string{"signup"}
				case "help":
					app.QueueUpdateDraw(func() {
						nextStepBox.SetText("[yellow]Helpful flow:[-] `signup <email>` or `login <token>` first.\nThen use `create-agent --agent <instance-id>`.\nThen launch chaos with `create-experiment --type cpu_stress --agent <id> --duration 30 --cpu 40`.")
					})
				case "create-agent":
					app.QueueUpdateDraw(func() {
						nextStepBox.SetText("[yellow]After create-agent:[-] run `create-experiment --type cpu_stress --agent <id> --duration 30 --cpu 40` or another supported experiment type.")
					})
				case "create-experiment":
					app.QueueUpdateDraw(func() {
						nextStepBox.SetText("[yellow]After create-experiment:[-] switch to Monitor to watch live metrics, or run another experiment against a different agent.")
					})
				default:
					app.QueueUpdateDraw(func() {
						nextStepBox.SetText("[yellow]Suggested next step:[-] if this command needs auth, run `signup <email>` or `login <token>` first.\n[gray]Otherwise:[-] try `help` to see the Lucifer command flow.")
					})
				}

				cmd := exec.Command("go", append([]string{"run", "main.go"}, args...)...)
				cmd.Dir = cliDir
				cmd.Env = append(os.Environ(), "CHAOS_SERVER_URL=http://localhost:8000")
				if stdinPayload != nil {
					cmd.Stdin = bytes.NewReader(stdinPayload)
				}
				out, err := cmd.CombinedOutput()

				app.QueueUpdateDraw(func() {
					if err != nil {
						fmt.Fprintf(termOutput, "[red]Execution Failure: %v[-]\n", err)
					}
					fmt.Fprintf(termOutput, "%s\n", string(out))
					termOutput.ScrollToEnd()
				})
			}(cmdText)
		}
	})

	luciferFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nextStepBox, 5, 0, false).
		AddItem(termOutput, 0, 1, false).
		AddItem(termInput, 3, 0, true)

	luciferFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' && termInput.GetText() == "" {
			nextState = "Quit"
			app.Stop()
			return nil
		}
		return event
	})

	pages.AddPage("1", luciferFlex, true, false)

	vaptView := report.GetVaptReportView()
	pages.AddPage("2", vaptView, true, false)

	tabBar.SetHighlightedFunc(func(added, _, _ []string) {
		if len(added) == 0 {
			return
		}
		active := added[0]
		tabBar.SetText(tabLabel(active))
		switch active {
		case "0":
			nextState = "Monitor"
			app.Stop()
		case "1":
			pages.SwitchToPage("1")
			app.SetFocus(termInput)
		case "2":
			pages.SwitchToPage("2")
			app.SetFocus(vaptView)
		case "3":
			nextState = "Quit"
			app.Stop()
		}
	})

	inner := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(tabBar, 4, 0, false).
		AddItem(pages, 0, 1, true).
		AddItem(notificationBar, 3, 0, false)

	root := tview.NewFlex().AddItem(nil, 0, 1, false).AddItem(inner, 120, 0, true).AddItem(nil, 0, 1, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.SetFocus(tabBar)
			return nil
		}
		if app.GetFocus() == tabBar {
			switch event.Rune() {
			case '1':
				tabBar.Highlight("0")
				return nil
			case '2':
				tabBar.Highlight("1")
				return nil
			case '3':
				tabBar.Highlight("2")
				return nil
			case '4', 'q':
				tabBar.Highlight("3")
				return nil
			}
		}
		return event
	})

	tabBar.Highlight("1")
	app.SetFocus(tabBar)

	if err := app.SetRoot(root, true).EnableMouse(true).Run(); err != nil {
		log.Fatalf("failed to run UI: %v", err)
	}
	return nextState
}
