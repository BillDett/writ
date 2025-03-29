package util

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

//go:embed gulliver.txt
var bigtext string

//go:embed asyoulik.txt
var mediumtext string

//go:embed warandpeace.txt
var war string

//go:embed proust.txt
var proust string

//go:embed ulysses.txt
var ulysses string

var base string
var pt *PieceTable
var result, answer string

func TestPieceTable(t *testing.T) {

	base = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	DeleteTests(t)

	EditTests(t)

	AuthorTest(t)

}

/*
Stress Test

Authoring...
Incrementally add all of the text for a few hundred thousand words, one character at a time. Introduce some occasional
backspaces. Test how well we handle thousands of tiny pieces.


Editing...
Take a big piece of text (a few hundred K).
Process a list of random edits (adds, deletions) from the piecetable
Compare the end result with the known result.

*/

func EditTests(t *testing.T) {
	fmt.Println("Insert at beginning of a piece")
	pt = NewPieceTable(base)
	pt.Insert(0, "FOO")
	result = pt.Text()
	answer = "FOO" + base
	if result != answer {
		t.Errorf("Fail: Beginning insert wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Insert at end of a piece")
	pt = NewPieceTable(base)
	pt.Insert(26, "FOO")
	result = pt.Text()
	answer = base + "FOO"
	if result != answer {
		t.Errorf("Fail: End insert wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Insert in middle of a piece")
	pt = NewPieceTable(base)
	pt.Insert(13, "FOO")
	result = pt.Text()
	answer = base[:13] + "FOO" + base[13:]
	if result != answer {
		t.Errorf("Fail: Middle insert wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Insert past end of buffer")
	pt = NewPieceTable(base)
	if pt.Insert(75, "FOO") {
		t.Errorf("Fail: Inserting past end of the buffer should have failed.\n")
	}

	fmt.Println("Multiple inserts")
	pt = NewPieceTable(base)
	pt.Insert(5, "FOO")
	pt.Insert(10, "BAR")
	pt.Insert(15, "123")
	pt.Insert(20, "456789")
	pt.Insert(7, "abc")
	result = pt.Text()
	answer = base[:5] + "FOabcO" + base[5:7] + "BAR" + base[7:9] + "123" + base[9:11] + "456789" + base[11:]
	if result != answer {
		t.Errorf("Fail: Middle insert  wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Delete across multiple pieces")
	pt.Delete(6, 9)
	result = pt.Text()
	answer = base[:6] + "R" + base[7:9] + "123" + base[9:11] + "456789" + base[11:]
	if result != answer {
		t.Errorf("Fail: Middle insert  wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Delete at beginning of a piece")
	pt = NewPieceTable(base)
	pt.Delete(0, 5)
	//pt.Dump()
	result = pt.Text()
	answer = base[5:]
	if result != answer {
		t.Errorf("Fail: Beginning delete wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Delete at end of a piece")
	pt = NewPieceTable(base)
	length := pt.size
	pt.Delete(length-5, 5)
	result = pt.Text()
	answer = base[:length-5]
	if result != answer {
		t.Errorf("Fail: End delete wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Delete in middle of a piece")
	pt = NewPieceTable(base)
	pt.Delete(13, 6)
	result = pt.Text()
	answer = base[:13] + base[19:]
	if result != answer {
		t.Errorf("Fail: Middle delete wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Add to Empty PieceTable")
	pt = NewPieceTable("")
	answer = "The quick brown fox jumped over the small dog."
	for i := 0; i < len(answer); i++ {
		pt.Insert(i, string(answer[i]))
	}
	result = pt.Text()
	if result != answer {
		t.Errorf("Fail: Append to empty, wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Create a large PieceTable from a huge string")
	pt = NewPieceTable("")
	pt.Insert(0, bigtext)
	result = pt.Text()
	if result != bigtext {
		t.Errorf("Fail: Create into empty bigtext, wanted >%s< got >%s<\n", answer, result)
	}
}

func DeleteTests(t *testing.T) {

	fmt.Println("Delete at beginning of a piece")
	pt = NewPieceTable(base)
	pt.Insert(8, "FOO")
	pt.DeleteRune(8)
	result = pt.Text()
	answer = "ABCDEFGHOOIJKLMNOPQRSTUVWXYZ"
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: Beginning delete wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: Beginning delete")
	}

	fmt.Println("Delete at end of a piece")
	pt = NewPieceTable(base)
	pt.Insert(8, "FOO")
	pt.DeleteRune(10)
	result = pt.Text()
	answer = "ABCDEFGHFOIJKLMNOPQRSTUVWXYZ"
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: End delete wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: End delete")
	}

	fmt.Println("Delete in middle of a piece")
	pt = NewPieceTable(base)
	pt.Insert(8, "FOA")
	pt.DeleteRune(9)
	result = pt.Text()
	answer = "ABCDEFGHFAIJKLMNOPQRSTUVWXYZ"
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: Middle delete wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: Middle delete")
	}

	fmt.Println("Delete in middle of a large piece")
	pt = NewPieceTable(base)
	pt.Insert(19, base)
	pt.DeleteRune(20)
	result = pt.Text()
	answer = "ABCDEFGHIJKLMNOPQRS" + "ACDEFGHIJKLMNOPQRSTUVWXYZ" + "TUVWXYZ"
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: Middle large delete wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: Middle large delete")
	}

	fmt.Println("Delete a single rune piece")
	pt = NewPieceTable(base)
	pt.Insert(8, "X")
	pt.DeleteRune(8)
	result = pt.Text()
	answer = base
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: Single rune piece delete wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: Single rune delete")
	}

	fmt.Println("Sequential Delete")
	pt = NewPieceTable(base)
	pt.DeleteRune(17) // R
	pt.DeleteRune(17) // S
	pt.DeleteRune(17) // T
	pt.DeleteRune(17) // U
	pt.DeleteRune(17) // V
	pt.DeleteRune(17) // W
	result = pt.Text()
	// R..W
	answer = "ABCDEFGHIJKLMNOPQXYZ"
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: Sequential delete wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: Sequential delete")
	}

	fmt.Println("Staggered Delete")
	pt = NewPieceTable(base)

	pt.DeleteRune(17) // ABCDEFGHIJKLMNOPQSTUVWXYZ
	pt.DeleteRune(19) // ABCDEFGHIJKLMNOPQSTVWXYZ
	pt.DeleteRune(21) // ABCDEFGHIJKLMNOPQSTVWYZ
	pt.DeleteRune(2)  // ABDEFGHIJKLMNOPQSTVWYZ
	pt.DeleteRune(4)  // ABDEGHIJKLMNOPQSTVWYZ
	pt.DeleteRune(6)  // ABDEGHJKLMNOPQSTVWYZ
	result = pt.Text()
	answer = "ABDEGHJKLMNOPQSTVWYZ"
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: Staggered delete wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: Staggered delete")
	}

	fmt.Println("Random Delete")
	base = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	pt = NewPieceTable(base)
	pt.Insert(12, "HELLO") // "ABCDEFGHIJKLMHELLONOPQRSTUVWXYZ"
	pt.Insert(0, "WORLD")  // "WORLDABCDEFGHIJKLMHELLONOPQRSTUVWXYZ"
	pt.Insert(9, "THE")    // "WORLDABCDETHEFGHIJKLMHELLONOPQRSTUVWXYZ"
	pt.Insert(22, "QUICK") // "WORLDABCDETHEFGHIJKLMHEQUICKLLONOPQRSTUVWXYZ"
	pt.DeleteRune(3)       // "WORDABCDTHEEFGHIJKLHEQUICKLLOMNOPQRSTUVWXYZ"
	pt.DeleteRune(23)      // "WORDABCDTHEEFGHIJKLHEQUCKLLOMNOPQRSTUVWXYZ"
	pt.DeleteRune(22)      // "WORDABCDTHEEFGHIJKLHEQCKLLOMNOPQRSTUVWXYZ"
	result = pt.Text()
	answer = "WORDABCDTHEEFGHIJKLHEQCKLLOMNOPQRSTUVWXYZ"
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: Random delete wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: Random delete")
	}

	fmt.Println("Delete outside of buffer")
	pt = NewPieceTable(base)
	if pt.DeleteRune(50) {
		t.Errorf("Fail: Delete outside buffer should not have succeeded!")
	} else {
		t.Logf("Success: Delete outside buffer was not allowed")
	}

	fmt.Println("Delete span of runes beyond buffer")
	pt = NewPieceTable(base)
	if pt.Delete(3, 50) {
		t.Errorf("Fail: Delete span beyond buffer should not have succeeded!")
	} else {
		t.Logf("Success: Delete span beyond buffer was not allowed")
	}

	fmt.Println("Delete span of runes")
	pt = NewPieceTable(base)
	if !pt.Delete(3, 15) { // "ABCSTUVWXYZ"
		t.Errorf("Fail: Delete span first delete did not succeed\n")
	}
	if !pt.Delete(4, 5) { // "ABCSYZ"
		t.Errorf("Fail: Delete span second delete did not succeed\n")
	}
	result = pt.Text()
	answer = "ABCSYZ"
	if strings.Compare(result, answer) != 0 {
		t.Errorf("Fail: Delete span wanted >%s< got >%s<\n", answer, result)
	} else {
		t.Logf("Success: Delete span")
	}
}

func AuthorTest(t *testing.T) {
	fmt.Println("Author Test")

	testString := war

	pt := NewPieceTable("")
	origsize := utf8.RuneCountInString(testString)
	for _, char := range testString {
		pt.AppendRune(char)
	}
	fmt.Printf("\tPieceTable has %d pieces\n", pt.NumPieces())
	newsize := utf8.RuneCountInString(pt.Text())
	if origsize != newsize {
		t.Errorf("Author fail- wanted %d, saw %d\n", origsize, newsize)
	} else {
		fmt.Printf("\tExpected %d runes, saw %d runes\n", origsize, newsize)
	}

	normalizedNew := norm.NFC.String(pt.Text())
	normalizedOrig := norm.NFC.String(testString)
	if normalizedNew != normalizedOrig {
		t.Errorf("Author fail- generated text does not match original\n")
		err := ioutil.WriteFile("author.txt", []byte(normalizedOrig), 0644)
		if err != nil {
			t.Error("Fail: Append test couldn't write to answer file")
		}
		err = ioutil.WriteFile("author_new.txt", []byte(normalizedNew), 0644)
		if err != nil {
			t.Error("Fail: Append test couldn't write to result file")
		}

	} else {
		fmt.Printf("\tGenerated text matches original\n")
	}
}
