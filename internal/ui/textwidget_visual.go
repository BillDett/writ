package ui

import (
	"fmt"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

//////// TextWidget Visual

// Clear out the widget using absolute screen coordinates
func (t *TextWidget) clear(screen tcell.Screen) {
	tx, ty, width, height := t.GetInnerRect()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			screen.SetContent(x+tx, y+ty, ' ', nil, t.style)
		}
	}
}

// layoutText recomputes the lineIndex array
// TODO: Ideally we don't layout the entire buffer every time- but rather just what is visible (or potentially visible) to the user. For large buffers this
//
//	could eliminate a lot of extra work.
func (t *TextWidget) layoutText() {
	t.lineIndex = nil
	row := 0
	t.runes = t.buffer.Runes()
	start := 0
	end := t.nextLine(start)
	for end != len(*t.runes)-1 {
		t.lineIndex = append(t.lineIndex, &linePair{start, end})
		start = end + 1
		end = t.nextLine(start)
		row++
	}
	t.lineIndex = append(t.lineIndex, &linePair{start, end})
}

// Draw renders visible buffer using t.currentLine and the t.lineIndex array
func (t *TextWidget) Draw(screen tcell.Screen) {
	t.Box.DrawForSubclass(screen, t)
	t.clear(screen)
	_, _, _, height := t.GetInnerRect()
	// TODO: We don't always need to layout the text with each call to Draw(), only when the text has changed. We should optimize this to be conditional based on a dirty flag.
	t.layoutText()
	row := 0
	for l := t.topLine; l < len(t.lineIndex) && row < height; l++ {
		t.drawLine(screen, t.lineIndex[l].start, t.lineIndex[l].end, row)
		row++
	}
	t.placeAndDrawCursor(screen)
}

// drawLine renders a single line of text from the buffer into the View from 'start' to 'end' inclusive
// using absolute screen coordinates
func (t *TextWidget) drawLine(screen tcell.Screen, start int, end int, y int) {
	if start == 0 && end == 0 { // nothing to draw
		return
	} else {
		tx, ty, _, _ := t.GetInnerRect()
		x := 0
		for c := start; c <= end; c++ {
			style := t.style
			//if t.IsSelecting() {
			if c >= t.selStart && c <= t.selEnd { // Are we drawing runes that are selected?
				style = t.selectedStyle
			}
			//}
			//t.view.SetContent(x, y, (*t.runes)[c], nil, style)
			screen.SetContent(x+tx, y+ty, (*t.runes)[c], nil, style)
			x += t.widthOf((*t.runes)[c])
		}
	}
}

// nextLine scans forward in the buffer from 'start' and returns the index of first character of next line or -1 if we are on last line of buffer
func (t *TextWidget) nextLine(start int) int {
	length := len(*t.runes)
	_, _, width, _ := t.GetInnerRect()
	column_count := 1
	for p := start; p < length; p++ {
		if (*t.runes)[p] == '\n' { // Found a newline, surely this denotes the end of a line
			if p < length {
				return p // There is more text "below" us, return this newline
			} else {
				return length - 1 // At end of buffer (and this takes up the whole line)
			}
		} else if column_count == width { // Reached the rightmost spot in the InnerRect
			if !unicode.IsSpace((*t.runes)[p]) { // Are we in the middle of a word- do we need to wordwrap?
				k := p
				for k > start && !unicode.IsSpace((*t.runes)[k]) { // "back up" until we find a space or the beginning of the line
					k--
				}
				if unicode.IsSpace((*t.runes)[k]) { // We found a whitespace, so return it
					return k
				} else { // We hit beginning of line, no whitespace at all, just cut the line where we originally found it
					return p
				}
			} else { // no need to split word, we're on a space already
				return p
			}
		} else {
			column_count += t.widthOf((*t.runes)[p])
		}
	}
	return length - 1 // If you get here, you went thru entire buffer w/out spanning a full line
}

// Determine the width (# of columns) for a particular rune
func (t *TextWidget) widthOf(r rune) int {
	w, found := runeWidths[r]
	if found {
		return w
	} else {
		return 1
	}
}

/*
placeAndDrawCursor() ensures that the cursor (if visible) is positioned correctly based on t.currentPosition.

	Set t.cursXPos, t.cursYPos and t.currentLine
*/
func (t *TextWidget) placeAndDrawCursor(screen tcell.Screen) {
	if t.HasFocus() { //t.cursorVisible { //&& !t.window.PopUpActive() {
		if len(t.lineIndex) > 0 {
			// Figure out what our currentLine ought to be based on currentPosition
			for l := 0; l < len(t.lineIndex); l++ {
				if t.lineIndex[l].start <= t.currentPosition && t.lineIndex[l].end >= t.currentPosition {
					t.currentLine = l
					break
				}
			}
			// Calculate the X position based on widths of all runes between start and currentPosition
			t.cursXPos = 0
			for w := t.lineIndex[t.currentLine].start; w < t.currentPosition; w++ {
				t.cursXPos += t.widthOf((*t.runes)[w])
			}
			t.cursYPos = t.currentLine - t.topLine
			// map from View to Screen coordinates
			tx, ty, _, _ := t.GetInnerRect()
			x := t.cursXPos + tx
			y := t.cursYPos + ty
			//t.window.screen.ShowCursor(x, y)
			screen.ShowCursor(x, y)
		}
	} else {
		//t.window.screen.HideCursor()
		screen.HideCursor()
	}
}

func (t *TextWidget) SetStyle(style tcell.Style) {
	t.style = style
}

func (t *TextWidget) SetSelectedStyle(style tcell.Style) {
	t.selectedStyle = style
}

func (t *TextWidget) CursorVisible(visible bool) {
	t.cursorVisible = visible
}

// Return the index where we are currently in the buffer (0..len(buffer)-1) as calculated via the cursor position
func (t *TextWidget) currentPositionFromCursor() int {
	if len(t.lineIndex) > 0 {
		return t.lineIndex[t.currentLine].start + t.cursXPos
	} else {
		return 0
	}
}

// additional draw function for TextWidget that further customizes the border
// Adheres to the requirement stated by tview.Box.SetDrawFunc()
func (t *TextWidget) text_draw(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	innerx := x + 1
	innery := y + 1
	innerw := width - 2
	innerh := height - 2
	bottom_border := height - 1
	style := tcell.StyleDefault.
		Background(tview.Styles.PrimitiveBackgroundColor).
		Foreground(tview.Styles.PrimaryTextColor)
	mod := ' '
	if t.dirty {
		mod = '*'
	}
	msg := fmt.Sprintf(" %c line: %d/%d  char: %d/%d  words: %d ", mod, t.currentLine+1, t.NumLines(),
		t.currentPosition+1, t.NumCharacters(), t.NumWords())
	startx := x + width - len(msg) - 1 // align right
	for i, r := range msg {
		screen.SetContent(startx+i, bottom_border, r, nil, style)
	}
	return innerx, innery, innerw, innerh
}
