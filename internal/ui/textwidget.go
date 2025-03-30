package ui

import (
	"fmt"
	"strings"
	"unicode"
	"writ/internal/util"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

//////// TextWidget

type TextWidget struct {
	//window        *MainWindow
	//view          views.View
	*tview.Box

	window *MainWindow

	style         tcell.Style
	selectedStyle tcell.Style

	currentDocKey string

	buffer *util.PieceTable
	runes  *[]rune // temporary array of 'computed' runes from PieceTable
	dirty  bool    // Has the buffer been changed?

	currentPosition int // The current position within the buffer	TODO: REMOVE THIS & JUST USE THE FUNCTION

	cursorVisible bool // Is cursor being shown?
	cursXPos      int  // Last cursor x position (in View coordinates)
	cursYPos      int  // Last cursor y position (in View coordinates)

	selStart  int // Index of selection start (or -1 if no selection)
	selEnd    int // Index of selection end (or -1 if no selection)
	selecting bool

	clipboard []rune // Temporary slice of runes from the buffer (for copy/cut/paste)

	currentFilePath string // Filepath to wherever the current Document has been saved to or read from

	topLine     int // Which line in lineIndex is topmost in view?
	currentLine int // What line in lineIndex is cursor currently on?
	lineIndex   []*linePair
}

// linePair is a tuple containing start/end indices for a display line
type linePair struct {
	start int // index of first character to render in line
	end   int // index of last character to render in line
}

// Define any runes that need more than 1 column (otherwise we default to 1 column)
// TODO: We should allow these to be overridden when we have some sort of settings/config
var runeWidths = map[rune]int{
	'\t': 4,
}

const bufferEnd string = "\ufeff" // Nonprinting character to allow us to append to the buffer more easily

func NewTextWidget() *TextWidget {
	tv := &TextWidget{
		Box: tview.NewBox().SetBorder(true),
	}

	tv.SetDrawFunc(tv.text_draw)

	tv.buffer = util.NewPieceTable(bufferEnd)
	tv.style = tcell.StyleDefault
	tv.dirty = false
	tv.ClearSelection()
	return tv
}

func (t *TextWidget) SetWindow(m *MainWindow) *TextWidget {
	t.window = m
	return t
}

func (t *TextWidget) IsSelecting() bool { return t.selecting }

func (t *TextWidget) ClearSelection() {
	t.selStart = -1
	t.selEnd = -1
	t.selecting = false
}

func (t *TextWidget) GetText() string {
	return strings.TrimSuffix(t.buffer.Text(), bufferEnd) // Drop the last nonprinting char we use in the editor
}

func (t *TextWidget) GetDocKey() string { return t.currentDocKey }

func (t *TextWidget) SetDocument(key string, name string, text string) {
	t.currentDocKey = key
	t.SetTitle(fmt.Sprintf(" %s ", name))
	t.SetText(text)
}

func (t *TextWidget) SetText(text string) {
	t.reset()
	t.buffer = util.NewPieceTable(bufferEnd)
	if text != "" {
		t.buffer.InsertRunes(0, []rune(text))
	}
}

func (t *TextWidget) SetBuffer(pt *util.PieceTable) {
	t.buffer = pt
}

func (t *TextWidget) reset() {
	t.cursXPos = 0
	t.cursYPos = 0
	t.currentPosition = 0
	t.currentLine = 0
	t.topLine = 0
	t.dirty = false
}

func (t *TextWidget) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return t.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		mod := event.Modifiers()
		switch event.Key() {
		case tcell.KeyDown:
			t.moveDown()
		case tcell.KeyUp:
			t.moveUp()
		case tcell.KeyRight:
			if mod == tcell.ModShift {
				t.moveRight(true)
			} else {
				t.moveRight(false)
			}
		case tcell.KeyLeft:
			if mod == tcell.ModShift {
				t.moveLeft(true)
			} else {
				t.moveLeft(false)
			}
		case tcell.KeyPgUp: // Fn+Up Arrow on MacOS
			t.pageUp()
		case tcell.KeyPgDn: // Fn+Down Arrow on MacOS
			t.pageDown()
		case tcell.KeyRune, tcell.KeyTAB:
			t.appendRune(event.Rune())
		case tcell.KeyHome:
			t.moveHome()
		case tcell.KeyEnd:
			t.moveEnd()
		case tcell.KeyEnter:
			t.enterPressed()
		case tcell.KeyBackspace, tcell.KeyBackspace2: // Delete on MacOS
			t.backspace()
		case tcell.KeyDelete: // Fn+Delete on MacOS
			t.delete()
		case tcell.KeyCtrlK: // Put us into selection mode
			if !t.IsSelecting() {
				t.startSelection()
			}
		case tcell.KeyCtrlS:
			if t.dirty {
				t.window.store.SaveDocument(t.currentDocKey, t.GetText())
				t.dirty = false
			}
		case tcell.KeyESC:
			if t.IsSelecting() {
				t.ClearSelection()
			}
		case tcell.KeyCtrlC:
			t.copySelection()
			t.ClearSelection()
		case tcell.KeyCtrlX:
			t.cutSelection()
			t.ClearSelection()
		case tcell.KeyCtrlV:
			t.pasteSelection()
		}
	})
}

// Dump produces a debug message for troubleshooting
func (t *TextWidget) Dump() string {
	buf := string(*t.runes)
	currentRune := '^'
	if len(*t.runes) > 0 {
		currentRune = (*t.runes)[t.currentPositionFromCursor()]
	}
	result := fmt.Sprintf(">%s<\nBuffer Length: %d, Current Line: %d, Current rune: %q, Current Position: %d, Top Line: %d, cursX: %d, cursY: %d, selStart: %d, selEnd: %d\n",
		buf, t.buffer.Length(), t.currentLine, currentRune, t.currentPositionFromCursor(), t.topLine, t.cursXPos, t.cursYPos, t.selStart, t.selEnd)
	result = fmt.Sprintf("%s\nLineIndex Length: %d\n", result, len(t.lineIndex))
	result = fmt.Sprintf("%s\nLineIndex:\n", result)
	for _, lp := range t.lineIndex {
		result = fmt.Sprintf("%s\n%+v", result, *lp)
	}
	return result
}

func (t *TextWidget) Buffer() *util.PieceTable { return t.buffer }

// Handy way for another component to see what's the current Rune in the Editor
func (t *TextWidget) CurrentRune() rune {
	currentRune := '^'
	if (t.runes != nil) && (len(*t.runes) > 0) && t.currentPosition < len(*t.runes) {
		currentRune = (*t.runes)[t.currentPosition]
	}
	return currentRune
}

func (t *TextWidget) NumCharacters() int {
	return t.buffer.Length()
}

func (t *TextWidget) NumLines() int {
	return len(t.lineIndex)
}

func (t *TextWidget) NumWords() int {
	inWord := false
	wordCount := 0

	for _, r := range *t.buffer.Runes() {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				wordCount++
				inWord = true
			}
		} else {
			inWord = false
		}
	}

	return wordCount
}

func (t *TextWidget) IsModified() bool {
	return t.dirty
}

func (t *TextWidget) Focus(delegate func(p tview.Primitive)) {
	t.window.SetLastFocused(t)
	t.Box.Focus(delegate)
}

func (t *TextWidget) SetModified(state bool) { t.dirty = state }

func (t *TextWidget) GetFilePath() string { return t.currentFilePath }

func (t *TextWidget) SetFilePath(fp string) { t.currentFilePath = fp }
