package ui

//////// TextWidget Navigation

/*

Cursor movement functions

These first and foremost make sure we are positioned correctly within the buffer, based on the intended movement operation. We also detect the need for scrolling and adjust t.topLine as necessary.
The actual X/Y coordinate of the cursor itself is handled during the Draw() method- this is because we need to do this consistently whether or not the cursor moves due to a navigation event
or the editing of text. This is possible because we can always re-compute our overall screen position from t.currentPosition and t.lineIndex entries.

*/

func (t *TextWidget) moveDown() {
	if t.currentLine != len(t.lineIndex)-1 { // If we're not on the last line...
		//previousPosition := t.currentPosition
		t.currentLine++
		// Place the cursor in same spot in new line, or at end if we were past it visually on previous line
		logicalEnd := t.lineIndex[t.currentLine].end - t.lineIndex[t.currentLine].start
		if t.cursXPos > logicalEnd {
			t.currentPosition = t.lineIndex[t.currentLine].end
		} else {
			t.currentPosition = t.lineIndex[t.currentLine].start + t.cursXPos
		}
		// Handle scrolling
		_, _, _, height := t.GetInnerRect()
		lastVisibleLine := t.topLine + height
		if t.currentLine == lastVisibleLine {
			t.topLine++
		}
		if t.IsSelecting() {
			if t.currentPosition > t.selEnd { // Extending right selection "down"
				t.selEnd = t.currentPosition
			} else {
				t.selStart = t.currentPosition // Shrinking left selection "down"
			}

		} else {
			t.ClearSelection()
		}
	}
}

func (t *TextWidget) moveUp() {
	if t.currentLine != 0 { // If we're not on the first line
		// Handle scrolling
		if t.currentLine == t.topLine {
			t.topLine--
		}
		t.currentLine--
		// Place the cursor in same spot in new line, or at end if we were past it visually on previous line
		logicalEnd := t.lineIndex[t.currentLine].end - t.lineIndex[t.currentLine].start
		if t.cursXPos > logicalEnd {
			t.currentPosition = t.lineIndex[t.currentLine].end
		} else {
			t.currentPosition = t.lineIndex[t.currentLine].start + t.cursXPos
		}
		if t.IsSelecting() {
			if t.currentPosition < t.selStart { // Extending left selection "up"
				t.selStart = t.currentPosition
			} else {
				t.selEnd = t.currentPosition // Shrinking right selection "up"
			}
		} else {
			t.ClearSelection()
		}

	}
}

func (t *TextWidget) moveRight(shifted bool) {
	if t.currentPosition != len(*t.runes)-1 { // If we not on the last char in buffer...
		t.currentPosition++
		if t.currentPosition > t.lineIndex[t.currentLine].end { // have we gone past logical line end?
			// Handle scrolling
			_, _, _, height := t.GetInnerRect()
			lastVisibleLine := t.topLine + height
			if t.currentLine+1 == lastVisibleLine {
				t.topLine++
			}
		}
		if shifted {
			if !t.IsSelecting() {
				t.selStart = t.currentPosition - 1
				t.selEnd = t.currentPosition - 1
				t.selecting = true
			} else {
				if t.currentPosition-1 == t.selStart { // Did we just move right from start of left selection?
					t.selStart = t.currentPosition // "shrink" start of left selection
				} else {
					t.selEnd = t.currentPosition - 1
				}
			}
		} else {
			t.ClearSelection()
		}
	}
}

/*
func (t *TextWidget) selectRight() {
	if t.currentPosition != len(*t.runes)-1 { // If we not on the last char in buffer...
		if !t.IsSelecting() {
			t.selStart = t.currentPosition
			t.selecting = true
		}
		t.currentPosition++
		if t.currentPosition > t.lineIndex[t.currentLine].end { // have we gone past logical line end?
			// Handle scrolling
			_, height := t.view.Size()
			lastVisibleLine := t.topLine + height
			if t.currentLine+1 == lastVisibleLine {
				t.topLine++
			}
		}
		if t.currentPosition > t.selEnd { // Extending right selection
			t.selEnd = t.currentPosition - 1
		} else {
			t.selStart = t.currentPosition - 1 // Shrinking left selection
		}
	}
}
*/

func (t *TextWidget) moveLeft(shifted bool) {
	if t.currentPosition != 0 { // We are not at start of buffer
		if t.currentPosition == t.lineIndex[t.currentLine].start { // If we are in first position of current line
			if t.currentLine != 0 {
				t.currentLine--
			}
		}
		t.currentPosition--
		// Handle scrolling
		if t.currentLine < t.topLine {
			t.topLine--
		}
		// TODO: Doesn't account for when we move 'past' start of selection- should flip
		if shifted {
			if !t.IsSelecting() {
				t.selStart = t.currentPosition
				t.selEnd = t.currentPosition
				t.selecting = true
			} else {
				if t.currentPosition == t.selEnd { // Did we just move left from end of right selection?
					t.selEnd = t.currentPosition - 1 // "shrink" right selection
				} else {
					t.selStart = t.currentPosition
				}
			}
		} else {
			t.ClearSelection()
		}
	}
}

/*
func (t *TextWidget) selectLeft() {
	if t.currentPosition != 0 { // We are not at start of buffer
		if !t.IsSelecting() {
			t.selStart = t.currentPosition - 1
			t.selEnd = t.currentPosition
			t.selecting = true
		}
		if t.currentPosition == t.lineIndex[t.currentLine].start { // If we are in first position of current line
			if t.currentLine != 0 {
				t.currentLine--
			}
		}
		t.currentPosition--
		// Handle scrolling
		if t.currentLine < t.topLine {
			t.topLine--
		}
		if t.currentPosition < t.selStart { // Extending left selection
			t.selStart = t.currentPosition
		} else {
			t.selEnd = t.currentPosition // Shrinking right selection
		}
	}
}
*/

func (t *TextWidget) pageDown() {
	_, _, _, height := t.GetInnerRect()
	lastVisibleLine := t.topLine + height - 1
	if len(t.lineIndex) > lastVisibleLine { // There are more lines below current View
		t.topLine = lastVisibleLine + 1
		t.currentLine = t.topLine
	}
	// Place the cursor in same spot in new line, or at end if we were past it visually on previous line
	logicalEnd := t.lineIndex[t.currentLine].end - t.lineIndex[t.currentLine].start
	if t.cursXPos > logicalEnd {
		t.currentPosition = t.lineIndex[t.currentLine].end
	} else {
		t.currentPosition = t.lineIndex[t.currentLine].start + t.cursXPos
	}
	if t.IsSelecting() {
		t.selEnd = t.currentPosition
	} else {
		t.ClearSelection()
	}
}

func (t *TextWidget) pageUp() {
	_, _, _, height := t.GetInnerRect()
	if t.topLine-height > 0 { // We have at least a View's worth above us
		t.topLine -= height
	} else {
		t.topLine = 0 // We have less than a View's worth above us, just jump to beginning
	}
	t.currentLine = t.topLine + t.cursYPos
	// Place the cursor in same spot in new line, or at end if we were past it visually on previous line
	logicalEnd := t.lineIndex[t.currentLine].end - t.lineIndex[t.currentLine].start
	if t.cursXPos > logicalEnd {
		t.currentPosition = t.lineIndex[t.currentLine].end
	} else {
		t.currentPosition = t.lineIndex[t.currentLine].start + t.cursXPos
	}
	if t.IsSelecting() {
		t.selEnd = t.currentPosition
	} else {
		t.ClearSelection()
	}
}

func (t *TextWidget) moveHome() {
	// TODO: Will need to make this smarter one we introduce tabs
	t.currentPosition = t.lineIndex[t.currentLine].start
	if t.IsSelecting() {
		if t.currentPosition < t.selStart { // Extending left selection
			t.selStart = t.currentPosition
		} else {
			t.selEnd = t.currentPosition // Shrinking right selection
		}

	} else {
		t.ClearSelection()
	}
}

func (t *TextWidget) moveEnd() {
	t.currentPosition = t.lineIndex[t.currentLine].end
	if t.IsSelecting() {
		if t.currentPosition > t.selEnd { // Extending right selection
			t.selEnd = t.currentPosition - 1
		} else {
			t.selStart = t.currentPosition - 1 // Shrinking left selection
		}
	} else {
		t.ClearSelection()
	}
}
