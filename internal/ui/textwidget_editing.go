package ui

import "github.com/atotto/clipboard"

//////// TextWidget Editing

func (t *TextWidget) appendRune(r rune) {
	t.buffer.InsertRunes(t.currentPosition, []rune{r})
	t.dirty = true
	t.currentPosition++
	// Handle scrolling
	_, _, _, height := t.GetInnerRect()
	lastVisibleLine := t.topLine + height
	if t.currentLine == lastVisibleLine {
		t.topLine++
	}
}

// enterPressed handles the logic when we insert a newline- and what we might need to do to scroll the position
func (t *TextWidget) enterPressed() {
	t.buffer.InsertRunes(t.currentPosition, []rune{'\n'})
	t.dirty = true
	t.currentPosition++
	t.currentLine++
	// Handle scrolling
	_, _, _, height := t.GetInnerRect()
	lastVisibleLine := t.topLine + height
	if t.currentLine == lastVisibleLine {
		t.topLine++
	}
}

func (t *TextWidget) backspace() {
	if t.currentPosition == 0 { // Do nothing if on first character
		return
	} else { // We are not on top leftmost position of screen
		if t.IsSelecting() { // Delete the selected block of text
			if t.selEnd != -1 { // Have we actually selected any runes?
				t.currentPosition = t.selStart - 1
				// Remove these runes from the buffer
				t.buffer.Delete(t.selStart, t.selEnd-t.selStart+1)
				t.ClearSelection()
			}
		} else { // Backspace from current position
			posToRemove := t.currentPosition - 1
			t.buffer.Delete(posToRemove, 1)
			if t.cursXPos == 0 && t.cursYPos == 0 { // Are we on top/leftmost position? Need to scroll up one line
				t.topLine--
				t.currentLine--
			}
			t.moveLeft(false)
		}
	}
}

func (t *TextWidget) delete() {
	if t.currentPosition != t.buffer.Length()-1 { // Never delete the last non-printing character of the buffer
		if t.IsSelecting() { // Delete the selected block of text
			if t.selEnd != -1 { // Have we actually selected any runes?
				t.currentPosition = t.selStart
				// Remove these runes from the buffer
				t.buffer.Delete(t.selStart, t.selEnd-t.selStart+1)
				t.ClearSelection()
			}
		} else { // Just delete current position
			t.buffer.Delete(t.currentPosition, 1)
		}
	}
}

func (t *TextWidget) startSelection() {
	t.selecting = true
	t.selStart = t.currentPosition
	// keep t.selEnd as -1...this means we started selecting but haven't added any runes yet
	// TODO: We should have a visual indicator that we're in selection mode in status bar
}

func (t *TextWidget) copySelection() bool {
	// Grab all of the runes from selStart to selEnd and save to the system clipboard
	if t.IsSelecting() {
		if t.selEnd != -1 { // Have we actually selected any runes?
			size := t.selEnd - t.selStart + 1
			text := make([]rune, size)
			for c := 0; c < size; c++ {
				text[c] = (*t.runes)[c+t.selStart]
			}
			// Write to system clipboard
			err := clipboard.WriteAll(string(text))
			if err != nil {
				t.window.Error("Failed to copy to clipboard: " + err.Error())
				return false
			}
			return true
		}
	}
	return false
}

func (t *TextWidget) cutSelection() {
	// Only proceed with cut if copy was successful
	if t.copySelection() {
		if t.IsSelecting() && t.selEnd != -1 {
			// Place where our current position ought to be following the cut
			t.currentPosition = t.selStart
			// Remove these runes from the buffer
			t.buffer.Delete(t.selStart, t.selEnd-t.selStart+1)
			t.dirty = true
		}
	}
}

func (t *TextWidget) pasteSelection() {
	// Get text from system clipboard
	text, err := clipboard.ReadAll()
	if err != nil {
		t.window.Error("Failed to paste from clipboard: " + err.Error())
		return
	}

	// Add all the runes from the clipboard at the current position
	if text != "" {
		t.buffer.InsertRunes(t.currentPosition, []rune(text))
		t.currentPosition += len(text)
		t.dirty = true
	}
	// TODO: HANDLE SCROLLING- CURSOR SHOULD BE PLACED AT END OF PASTED TEXT- IF WE'RE OFF THE SCREEN
	//    THEN WE SHOULD SCROLL TO LAST THIRD OF PAGE
}
