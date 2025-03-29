package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

/*
	Piece Table implementation for the text editor- allows for efficient insert/delete activity on a sequence of runes

	TODO:
	* Add a stack for undo/redo actions against the PieceTable (hopefully this is actually pretty trivial)
		* Undo "pushes" last piece from piece table onto a stack
		* Redo "pops" the topmost piece from the stack back into the piece table
	* Add a method to get the rune at a position in the buffer

*/

type piece struct {
	source *[]rune // should point to original or add slices in piecetable
	start  int
	length int // Sum of all piece lengths == length of final edited text
}

// PieceTable manages efficient edits to a string of text
type PieceTable struct {
	original []rune
	add      []rune
	pieces   []piece
	size     int
}

// NewPieceTable creates a piecetable instance
func NewPieceTable(orig string) *PieceTable {
	origrunes := []rune(orig)
	length := len(origrunes)
	pt := PieceTable{
		origrunes,
		[]rune{},
		[]piece{},
		length,
	}
	p := piece{&(pt.original), 0, len(pt.original)}
	pt.pieces = append(pt.pieces, p)
	return &pt
}

// Dump generates a debug view of the PieceTable for troubleshooting
func (p *PieceTable) Dump() {
	fmt.Printf("\nOriginal buffer:\n\t(%p), %s\n", &p.original, string(p.original))
	fmt.Printf("Add buffer:\n\t(%p) %s\n", &p.add, string(p.add))
	fmt.Printf("Assembled:\n\t%s\n", p.Text())
	fmt.Printf("Size: %d\n", p.size)
	fmt.Printf("Pieces:\nSource\t\t\tStart\tLength\tSpan\n------\t\t\t-----\t------\t----------\n")
	for _, piece := range p.pieces {
		span := (*piece.source)[piece.start : piece.start+piece.length]
		fmt.Printf("%p\t\t%d\t%d\t%s\n", piece.source, piece.start, piece.length, string(span))
	}

	fmt.Println()
}

// Insert puts fragment into the string at given position
func (p *PieceTable) Insert(position int, fragment string) bool {
	fragrunes := []rune(fragment)
	return p.InsertRunes(position, fragrunes)
}

// InsertRunes puts a slice of runes into the string at given position
func (p *PieceTable) InsertRunes(position int, runes []rune) bool {

	if position <= p.size {
		// save in the add buffer and create the necessary piece instance
		start := len(p.add)
		length := len(runes)
		p.add = append(p.add, runes...)
		newadd := piece{&(p.add), start, length}

		//fmt.Printf("Inserting at position %d\n", position)

		if position == 0 {
			// insert newadd to front of pieces list
			p.pieces = append([]piece{newadd}, p.pieces...)
		} else if position == p.size {
			// append newadd to end of pieces list
			p.pieces = append(p.pieces, newadd)
		} else {
			// We have to look for the right piece now to split so we can insert mid-string
			totalLength := 0
			i := 0
			for i < len(p.pieces) {
				totalLength += p.pieces[i].length
				if totalLength >= position {
					break
				}
				i++
			}
			// We're on the piece that needs to be split
			newRemainder := (totalLength - position)      // What is length of the remainder after we split this piece?
			p.pieces[i].length -= newRemainder            // "shrink" this piece where we split it
			p.pieces = insertPiece(p.pieces, newadd, i+1) // Insert a piece for the thing we're inserting after split
			p.pieces = insertPiece(p.pieces,
				piece{p.pieces[i].source, p.pieces[i].start + p.pieces[i].length, newRemainder}, i+2) // Insert new piece (which we split from p.pieces[i]) for remainder
		}

		p.size += length

		return true
	} else {
		//fmt.Printf("Error: trying to add at position %d which is outside buffer size %d\n", position, p.size)
		return false
	}
}

// AppendRune will add a single rune to the end of the PieceTable
func (p *PieceTable) AppendRune(r rune) bool {
	return p.InsertRunes(p.size, []rune{r})
}

// Append will add the characters to the end of the PieceTable
func (p *PieceTable) Append(fragment string) bool {
	return p.Insert(p.size, fragment)
}

// Delete the rune at position, return whether or not you were successful
func (p *PieceTable) DeleteRune(position int) bool {
	//p.Dump()
	//fmt.Printf("\tDeleting position %d from buffer\n", position)
	// Locate the piece which contains the rune at position
	if position < p.size { // make sure we're within the buffer itself
		totalLength := 0 // total 'length' that we've seen while scanning for the piece where delete should happen
		idx := 0         // index of the piece where delete should happen
		for i, piece := range p.pieces {
			idx = i
			totalLength += piece.length
			if totalLength > position {
				break
			}
		}

		//fmt.Printf("\tFound delete point in piece %d at totalLength %d with start %d and length %d\n", idx, totalLength, p.pieces[idx].start, p.pieces[idx].length)

		// Remove the rune
		if (totalLength - p.pieces[idx].length) == position { // position falls on start of piece
			p.pieces[idx].start += 1 // "skip" this rune
			p.pieces[idx].length -= 1
			//fmt.Printf("\tDelete on start of piece, start is now %d\n", p.pieces[idx].start)
		} else if (totalLength - 1) == position { // position falls on end of piece
			p.pieces[idx].length -= 1 // "skip" this rune
			//fmt.Printf("\tDelete on end of piece, length is now %d\n", p.pieces[idx].length)
		} else { // position falls in middle of piece, need to split it up
			origLength := p.pieces[idx].length
			p.pieces[idx].length -= (totalLength - position) // 'trim' the current piece at deletion
			//fmt.Printf("\tDelete in middle of piece, left length is %d\n", p.pieces[idx].length)
			// Add a new piece next to this one with 'remainder' after the deleted rune
			newStart := p.pieces[idx].start + p.pieces[idx].length + 1 // find where we should pick up again
			newLength := origLength - p.pieces[idx].length - 1         // find difference in size, account for removed character
			p.pieces = insertPiece(p.pieces, piece{p.pieces[idx].source, newStart, newLength}, idx+1)
			//fmt.Printf("\tAdded new piece at index %d with start %d and length %d\n", idx+1, p.pieces[idx+1].start, p.pieces[idx+1].length)
		}

		p.size -= 1

		//p.Dump()
		return true
	} else {
		//fmt.Printf("\tDelete position %d given is outside size of buffer size %d\n", position, p.size)
		return false
	}
}

// Delete removes length characters starting at position, indicate if it was successful or not
// Not the most efficient way to do this, but it makes the code so much simpler
// Only big downside is undo actions are a bit more clumsy as they need to be reversed one rune at a time
func (p *PieceTable) Delete(position int, spanLength int) bool {
	//fmt.Printf("\tDeleting at position %d for %d runes\n", position, spanLength)
	if position+spanLength <= p.size {
		for d := 0; d < spanLength; d++ {
			if !p.DeleteRune(position) {
				return false
			}
		}
		return true
	} else {
		//fmt.Printf("\tRequested span of %d starting at %d goes beyond size of buffer %d\n", spanLength, position, p.size)
		return false
	}
}

// Text returns the string being managed by the PieceTable with all edits applied
func (p *PieceTable) Text() string {
	runes := p.Runes()
	return string(*runes)
}

// Runes returns the runes being managed by the PieceTable with all edits applied
func (p *PieceTable) Runes() *[]rune {
	var runes []rune
	for _, piece := range p.pieces {
		if piece.length != 0 {
			span := (*piece.source)[piece.start : piece.start+piece.length]
			runes = append(runes, span...)
		}
	}
	return &runes
}

func (p *PieceTable) Length() int {
	return p.size
}

func (p *PieceTable) NumPieces() int {
	return len(p.pieces)
}

// MarshalJSON is a custom marshaller so our PieceTable can be exported as a string of text
// TODO: Might be interesting to see about marshalling the PieceTable as a full data structure- so you could
// //	reconstruct the pieces (e.g. undo stack) directly.
func (p *PieceTable) MarshalJSON() ([]byte, error) {
	str := p.Text()
	str = strings.Replace(str, "\"", "\\\"", -1)
	result := "{ \"text\": \"" + str + "\" }"
	return []byte(result), nil
}

// UnmarshalJSON is a custom unmarshaller so a JSON string can be imported as a PieceTable
// Assumes JSON is of form { "text": "some string to initialize our PieceTable" }
// //	(if multiple entries exist in the JSON object, it is indeterminate which will be used for PieceTable)
func (p *PieceTable) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	// Initialize PieceTable with first value we get from map
	for _, value := range v.(map[string]interface{}) {
		p.Insert(0, value.(string)) // type coerce what we got into a string
		break
	}
	return nil
}

func insertPiece(slice []piece, newpiece piece, index int) []piece {
	s := append(slice, piece{})  // Making space for the new element
	copy(s[index+1:], s[index:]) // Shifting elements
	s[index] = newpiece          // Copying/inserting the value
	return s
}
