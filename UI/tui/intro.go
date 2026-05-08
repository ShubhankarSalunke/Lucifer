package tui

import (

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const luciferASCII = ` 		 _____            _____                    _____                    _____                    _____                    _____                    _____          
         /\    \          /\    \                  /\    \                  /\    \                  /\    \                  /\    \                  /\    \         
        /::\____\        /::\____\                /::\    \                /::\    \                /::\    \                /::\    \                /::\    \        
       /:::/    /       /:::/    /               /::::\    \               \:::\    \              /::::\    \              /::::\    \              /::::\    \       
      /:::/    /       /:::/    /               /::::::\    \               \:::\    \            /::::::\    \            /::::::\    \            /::::::\    \      
     /:::/    /       /:::/    /               /:::/\:::\    \               \:::\    \          /:::/\:::\    \          /:::/\:::\    \          /:::/\:::\    \     
    /:::/    /       /:::/    /               /:::/  \:::\    \               \:::\    \        /:::/__\:::\    \        /:::/__\:::\    \        /:::/__\:::\    \    
   /:::/    /       /:::/    /               /:::/    \:::\    \              /::::\    \      /::::\   \:::\    \      /::::\   \:::\    \      /::::\   \:::\    \   
  /:::/    /       /:::/    /      _____    /:::/    / \:::\    \    ____    /::::::\    \    /::::::\   \:::\    \    /::::::\   \:::\    \    /::::::\   \:::\    \  
 /:::/    /       /:::/____/      /\    \  /:::/    /   \:::\    \  /\   \  /:::/\:::\    \  /:::/\:::\   \:::\    \  /:::/\:::\   \:::\    \  /:::/\:::\   \:::\____\ 
/:::/____/       |:::|    /      /::\____\/:::/____/     \:::\____\/::\   \/:::/  \:::\____\/:::/  \:::\   \:::\____\/:::/__\:::\   \:::\____\/:::/  \:::\   \:::|    |
\:::\    \       |:::|____\     /:::/    /\:::\    \      \::/    /\:::\  /:::/    \::/    /\::/    \:::\   \::/    /\:::\   \:::\   \::/    /\::/   |::::\  /:::|____|
 \:::\    \       \:::\    \   /:::/    /  \:::\    \      \/____/  \:::\/:::/    / \/____/  \/____/ \:::\   \/____/  \:::\   \:::\   \/____/  \/____|:::::\/:::/    / 
  \:::\    \       \:::\    \ /:::/    /    \:::\    \               \::::::/    /                    \:::\    \       \:::\   \:::\    \            |:::::::::/    /  
   \:::\    \       \:::\    /:::/    /      \:::\    \               \::::/____/                      \:::\____\       \:::\   \:::\____\           |::|\::::/    /   
    \:::\    \       \:::\__/:::/    /        \:::\    \               \:::\    \                       \::/    /        \:::\   \::/    /           |::| \::/____/    
     \:::\    \       \::::::::/    /          \:::\    \               \:::\    \                       \/____/          \:::\   \/____/            |::|   |          
      \:::\    \       \::::::/    /            \:::\    \               \:::\    \                                        \:::\    \                |::|   |          
       \:::\____\       \::::/    /              \:::\____\               \:::\____\                                        \:::\____\               \::|   |          
         \::/    /        \::/	/                \::/    /                \::/    /                                         \::/    /                \:|   |           
         \/____/          \/____/                  \/____/                  \/____/                                           \/____/                  \|___|          
                                                                                                                                                                       `

func NewIntroScreen(app *tview.Application, onStart func()) tview.Primitive {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBackgroundColor(TrueBlack)

	content := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("\n[red::b]" + luciferASCII + "[-]\n\n" +
			"[white::b]CHAOS ENGINEERING FRAMEWORK[-]\n\n" +
			"[gray][ [green]SPACE[-][gray] ] start   [ [red]Q[-][gray] ] quit[-]\n")
	content.SetBackgroundColor(TrueBlack)

	spacer := tview.NewBox().SetBackgroundColor(TrueBlack)
	spacerBottom := tview.NewBox().SetBackgroundColor(TrueBlack)

	flex.
		AddItem(spacer, 0, 1, false).
		AddItem(content, 28, 0, false).
		AddItem(spacerBottom, 0, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
			onStart()
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			app.Stop()
		}
		return event
	})

	return flex
}
