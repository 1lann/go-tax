// Package dispenser creates sentence and word dispensers for text processing.
package dispenser

import (
	"strconv"
	"strings"
)

// A Dispenser dispenses words, sentences and numerals from an array of words,
// or a sentence.
type Dispenser struct {
	i                    int
	continueNextSentence bool
	str                  []string
}

// NewDispenser returns a new word dispenser.
func NewDispenser(str []string) *Dispenser {
	return &Dispenser{0, false, str}
}

// NewDispenserFromSentence returns a new word dispenser from a single sentence.
func NewDispenserFromSentence(str string) *Dispenser {
	strArr := strings.Split(str, " ")
	for k, word := range strArr {
		strArr[k] = word + " "
	}
	lastWord := strArr[len(strArr)-1]
	strArr[len(strArr)-1] = lastWord[:len(lastWord)-1]
	return &Dispenser{0, false, strArr}
}

func getNumeral(str string) (float64, bool) {
	str = strings.Replace(str, ",", "", -1)
	clean := strings.TrimLeft(strings.TrimSpace(str), "$")
	num, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, false
	}

	return num, true
}

// LastWord returns the previous word, but does not change the position.
func (d *Dispenser) LastWord() string {
	if d.i-2 < 0 || d.i-2 >= len(d.str) {
		return ""
	}

	return d.str[d.i-2]
}

// LastNWords returns the last N words, but does not change the position.
func (d *Dispenser) LastNWords(n int) string {
	if d.i-n-1 < 0 || d.i-n-1 >= len(d.str) {
		return ""
	}

	return d.str[d.i-2]
}

// NextWord returns whether or not there's another text word in the current sentence,
// and jumps to that word.
func (d *Dispenser) NextWord() bool {
	if d.i >= len(d.str) {
		return false
	}

	if !d.continueNextSentence {
		return false
	}

	_, ok := getNumeral(d.str[d.i])
	if ok {
		return false
	}

	if d.str[d.i][len(d.str[d.i])-1] != ' ' || d.i+1 == len(d.str) {
		d.continueNextSentence = false
	}

	d.i++

	return true
}

// Word returns the current word.
func (d *Dispenser) Word() string {
	if d.i > len(d.str) || d.i == 0 {
		return ""
	}

	return strings.TrimSpace(d.str[d.i-1])
}

// NextSentence jumps to the next sentence, and returns whether or not it's
// available.
func (d *Dispenser) NextSentence() bool {
	if d.i >= len(d.str) {
		return false
	}

	if !d.continueNextSentence {
		d.continueNextSentence = true
	} else {
		for ; d.i < len(d.str); d.i++ {
			if d.str[d.i][len(d.str[d.i])-1] != ' ' {
				d.i++
				break
			}
		}

		if d.i+1 == len(d.str) {
			d.i++
			return false
		} else if d.i >= len(d.str) {
			return false
		}
	}

	return true
}

// StartOfSentence jumps to the start of the current sentence.
func (d *Dispenser) StartOfSentence() {
	if d.i > 0 && d.i < len(d.str) &&
		d.str[d.i][len(d.str[d.i])-1] != ' ' &&
		d.str[d.i-1][len(d.str[d.i-1])-1] != ' ' {
		// Single word sentence
		if !d.continueNextSentence {
			d.i--
			d.continueNextSentence = true
			return
		}
		return
	}

	if d.i == 0 {
		d.continueNextSentence = true
		return
	}

	if len(d.str) == 1 {
		d.i = 0
		return
	}

	if d.i == len(d.str) || !d.continueNextSentence {
		d.i -= 2
	}

	if d.i < 0 {
		d.i = 0
	}

	d.continueNextSentence = true

	for ; d.i > 0; d.i-- {
		if d.str[d.i][len(d.str[d.i])-1] != ' ' {
			d.i++
			return
		}
	}
}

// LastSentence jumps to the start of the last sentence.
func (d *Dispenser) LastSentence() {
	d.StartOfSentence()

	if d.str[d.i][len(d.str[d.i])-1] != ' ' {
		// Also the end
		d.i--
		return
	}

	d.i -= 2
	if d.i <= 0 {
		d.i = 1
		return
	}
	d.StartOfSentence()
}

// DumpSentence returns the entire current sentence from the current position
// as a string, including numerals.
func (d *Dispenser) DumpSentence() string {
	if !d.continueNextSentence {
		return ""
	}

	sentence := ""
	for ; d.i < len(d.str); d.i++ {
		sentence += d.str[d.i]
		if d.str[d.i][len(d.str[d.i])-1] != ' ' {
			d.i++
			break
		}
	}

	d.continueNextSentence = false

	return sentence
}

// DumpNSentences returns the next N sentences without changing the position.
func (d *Dispenser) DumpNSentences(n int) []string {
	start := d.i
	continueState := d.continueNextSentence

	sentences := []string{d.DumpSentence()}
	for i := 0; i < n-1; i++ {
		if !d.NextSentence() {
			break
		}

		sentences = append(sentences, d.DumpSentence())
	}

	d.i = start
	d.continueNextSentence = continueState

	return sentences
}

// AtEndOfSentence returns whether or not the end of the sentence has been reached.
// It is theoretically the equivelant to NextWord() and NextNumeral() returning false.
func (d *Dispenser) AtEndOfSentence() bool {
	if !d.continueNextSentence {
		return true
	}

	if d.i >= len(d.str) {
		return true
	}

	return false
}

// NextNumeral returns whether or not the next word in the current sentence is
// a numeral, and jumps to the word.
func (d *Dispenser) NextNumeral() bool {
	if d.i >= len(d.str) {
		return false
	}

	if !d.continueNextSentence {
		return false
	}

	_, ok := getNumeral(d.str[d.i])
	if ok {
		if d.str[d.i][len(d.str[d.i])-1] != ' ' {
			d.continueNextSentence = false
		}

		d.i++

		return true
	}

	return false
}

// Position returns the current position in the text.
func (d *Dispenser) Position() int {
	return d.i
}

// JumpNextNumeral jumps to the next numeral in the sentence and returns true,
// or if there is no other numeral in the sentence, it does not jump at all and
// returns false.
func (d *Dispenser) JumpNextNumeral() bool {
	start := d.i

	for !d.AtEndOfSentence() {
		if d.NextNumeral() {
			d.i--
			d.continueNextSentence = true
			break
		} else {
			d.NextWord()
		}
	}

	hasNext := d.NextNumeral()
	if !hasNext && d.AtEndOfSentence() {
		d.i = start
		return false
	}

	d.continueNextSentence = true
	return true
}

// Numeral returns the current numeral.
//
// Examples of numerals: 4, 3.4, $4.50.
func (d *Dispenser) Numeral() float64 {
	if d.i > len(d.str) || d.i == 0 {
		return 0
	}

	num, ok := getNumeral(d.str[d.i-1])
	if ok {
		return num
	}

	return 0
}
