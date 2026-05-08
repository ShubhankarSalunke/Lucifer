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
	"cli/api"
	"cli/tui"

	"lucifer-cli/config"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	history      []string
	historyIndex int
)

func initToken() {
	tok := config.GetToken()
	if tok != "" {
		api.SetAuthToken(tok)
	}
}

func main() {
	initToken()

	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println(`
		CHAOS CONTROL TERMINAL - Help

		Usage:
		go run cmd/main.go [--help]
		make ui-help

		Navigation:
		Press Esc        Return to tab navigation bar
		Press Tab        Cycle through dashboard tabs
		Press q          Exit
		`)
		os.Exit(0)
	}

	state := "Intro"
	for {
		switch state {
		case "Intro":
			state = tui.RunIntro()
		case "Monitor":
			state = tui.RunTermUIMonitor()
		case "Main":
			state = runCLI()
		case "Quit":
			return
		default:
			state = "Intro"
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
	tview.Styles.PrimitiveBackgroundColor = tui.TrueBlack
	tview.Styles.ContrastBackgroundColor = tui.TrueBlack
	tview.Styles.MoreContrastBackgroundColor = tui.TrueBlack
	tview.Styles.BorderColor = tcell.NewRGBColor(30, 60, 90)
	tview.Styles.TitleColor = tcell.NewRGBColor(150, 150, 180)
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorSilver
	tview.Styles.TertiaryTextColor = tcell.ColorGray

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
	termOutput.SetBorder(true).SetTitle(" CHAOS SESSION ").SetTitleColor(tcell.ColorGreen).SetBackgroundColor(tui.TrueBlack)
	fmt.Fprintf(termOutput, "[gray]CHAOS CLI ready.[-] Type [white]signup <email>[-] to register, or [white]login <token>[-] if you already have a token.\n\n")

	nextStepBox := tview.NewTextView().SetDynamicColors(true).
		SetWrap(true)
	nextStepBox.SetBorder(true).
		SetTitle(" SUGGESTED STEPS ").
		SetTitleColor(tcell.ColorYellow).
		SetBackgroundColor(tui.TrueBlack)
	nextStepBox.SetText("[yellow]Start here:[-] `signup <email>` if you are new, or `login <token>` if you already have a Chaos API token.\n[gray]After auth:[-] try `create-agent --agent <instance-id>` or `create-experiment --type cpu_stress --agent <id> --duration 30 --cpu 40`.")

	termInput := tview.NewInputField().SetLabel("[green]> [-]").SetFieldWidth(0)
	termInput.SetBorder(true).SetTitle(" COMMANDS ").SetTitleColor(tcell.ColorGreen)
	termInput.SetBackgroundColor(tui.TrueBlack)

	termInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp {
			if historyIndex > 0 {
				historyIndex--
				termInput.SetText(history[historyIndex])
			}
			return nil
		}
		if event.Key() == tcell.KeyDown {
			if historyIndex < len(history)-1 {
				historyIndex++
				termInput.SetText(history[historyIndex])
			} else {
				historyIndex = len(history)
				termInput.SetText("")
			}
			return nil
		}
		return event
	})

	resolveLuciferBin := func() (string, error) {
		if bin := "./lucifer"; func() bool { _, err := os.Stat(bin); return err == nil }() {
			abspath, _ := filepath.Abs(bin)
			return abspath, nil
		}

		if self, err := os.Executable(); err == nil {
			if bin := filepath.Join(filepath.Dir(self), "lucifer"); func() bool { _, err := os.Stat(bin); return err == nil }() {
				return bin, nil
			}
		}

		return exec.LookPath("lucifer")
	}

	termInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			cmdText := termInput.GetText()
			if cmdText == "" {
				return
			}
			history = append(history, cmdText)
			historyIndex = len(history)
			termInput.SetText("")

			fmt.Fprintf(termOutput, "[white::b]> %s[-] \n", cmdText)

			go func(c string) {
				args := strings.Fields(c)
				if args[0] == "lucifer" {
					args = args[1:]
				}
				if len(args) == 0 {
					return
				}

				stdinPayload := []byte(nil)
				switch args[0] {
				case "login":
					if len(args) < 2 {
						app.QueueUpdateDraw(func() {
							fmt.Fprintf(termOutput, "[yellow]Usage: login <token>[-]\n")
							termOutput.ScrollToEnd()
						})
						return
					}
					stdinPayload = []byte(args[1] + "\n")
					args = []string{"login"}
				case "signup":
					userID := ""
					if len(args) > 1 {
						userID = strings.Join(args[1:], " ")
					}
					stdinPayload = []byte(userID + "\n")
					args = []string{"signup"}
				}

				
				cliMain := "main.go"
				cliPath := "../cli"
				var command *exec.Cmd
				if _, err := os.Stat(filepath.Join(cliPath, cliMain)); err == nil {
					command = exec.Command("go", "run", cliMain)
					command.Dir = cliPath
				} else {
					binPath, _ := resolveLuciferBin()
					command = exec.Command(binPath)
				}
				
				command.Args = append(command.Args, args...)

				command.Env = append(os.Environ(), "CHAOS_SERVER_URL=http://localhost:8000")
				if stdinPayload != nil {
					command.Stdin = bytes.NewReader(stdinPayload)
				}
				out, err := command.CombinedOutput()

				app.QueueUpdateDraw(func() {
					if err != nil {
						fmt.Fprintf(termOutput, "[red]Error: %v[-]\n", err)
					}
					fmt.Fprintf(termOutput, "%s\n", string(out))
					termOutput.ScrollToEnd()

					// RELOAD TOKEN
					tok := config.GetToken()
					if tok != "" {
						api.SetAuthToken(tok)
						notificationBar.SetText(fmt.Sprintf("[green]Auth Refreshed[-] | Token: %s...", tok[:8]))
					}
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

	pages.AddPage("1", luciferFlex, true, true)
	app.SetFocus(termInput)

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
		
		focus := app.GetFocus()
		if focus != nil {
			switch focus.(type) {
			case *tview.InputField, *tview.DropDown, *tview.Form, *tview.TextArea:
				return event
			}
		}

		if event.Key() == tcell.KeyTab {
			current := tabBar.GetHighlights()
			if len(current) > 0 {
				next := "0"
				switch current[0] {
				case "0": next = "1"
				case "1": next = "2"
				case "2": next = "3"
				case "3": next = "0"
				}
				tabBar.Highlight(next)
			}
			return nil
		}

		switch event.Rune() {
		case 'q':
			if app.GetFocus() != termInput {
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
