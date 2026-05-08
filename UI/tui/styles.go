package tui
import "github.com/gdamore/tcell/v2"
var TrueBlack = tcell.NewRGBColor(0, 0, 0)

var (
	ColorBackground = TrueBlack
	ColorPanelBg    = TrueBlack
	ColorHeaderBg   = TrueBlack

	ColorBorder    = tcell.NewRGBColor(30, 60, 90)
	ColorSteelBlue = tcell.ColorSteelBlue
	ColorGreen     = tcell.ColorGreen
	ColorYellow    = tcell.ColorYellow
	ColorAqua      = tcell.ColorAqua
	ColorRed       = tcell.ColorRed
	ColorOrange    = tcell.ColorOrange
	ColorPurple    = tcell.NewRGBColor(180, 120, 255)
)

const (
	vaptReportID = "RPT-2024-07"
)
