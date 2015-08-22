package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"rsc.io/pdf"
)

type transaction struct {
	Date   string
	Amount float64
	Memo   string
}

type handler interface {
	start(chan transaction)
	handleWord(word *pdf.Text, page int) bool
}

var h handler

func main() {
	substitutionsFile := flag.String("s", "", "substitutions file")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "usage: pdf2csv file.pdf [-s substitutions.csv]")
		os.Exit(2)
	}
	r, err := pdf.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	var substitutions Substitutions
	if *substitutionsFile != "" {
		if substitutions, err = NewSubstitutions(*substitutionsFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
	ch := make(chan transaction)
	go func() {
		h.start(ch)
	PageLoop:
		for i := 1; i <= r.NumPage(); i++ {
			words := findWords(r.Page(i).Content().Text)
			for _, t := range words {
				if h.handleWord(&t, i) {
					break PageLoop
				}
			}
		}
		close(ch)
	}()
	for t := range ch {
		account, amount, memo := substitutions.ReplaceData(t.Amount, t.Memo)
		fmt.Printf("\"%v:%v\",%v,\"%v\"\n", account, amount, t.Date, memo)
	}
}

func findWords(chars []pdf.Text) (words []pdf.Text) {
	// Sort by Y coordinate and normalize.
	const nudge = 1
	sort.Sort(pdf.TextVertical(chars))
	old := -100000.0
	for i, c := range chars {
		if c.Y != old && math.Abs(old-c.Y) < nudge {
			chars[i].Y = old
		} else {
			old = c.Y
		}
	}

	// Sort by Y coordinate, breaking ties with X.
	// This will bring letters in a single word together.
	sort.Sort(pdf.TextVertical(chars))

	// Loop over chars.
	for i := 0; i < len(chars); {
		// Find all chars on line.
		j := i + 1
		for j < len(chars) && chars[j].Y == chars[i].Y {
			j++
		}
		var end float64
		// Split line into words (really, phrases).
		for k := i; k < j; {
			ck := &chars[k]
			s := ck.S
			end = ck.X + ck.W
			charSpace := ck.FontSize / 6
			wordSpace := ck.FontSize * 2 / 3
			l := k + 1
			for l < j {
				// Grow word.
				cl := &chars[l]
				if sameFont(cl.Font, ck.Font) && math.Abs(cl.FontSize-ck.FontSize) < 0.1 && cl.X <= end+charSpace {
					s += cl.S
					end = cl.X + cl.W
					l++
					continue
				}
				// Add space to phrase before next word.
				if sameFont(cl.Font, ck.Font) && math.Abs(cl.FontSize-ck.FontSize) < 0.1 && cl.X <= end+wordSpace {
					s += " " + cl.S
					end = cl.X + cl.W
					l++
					continue
				}
				break
			}
			f := ck.Font
			f = strings.TrimSuffix(f, ",Italic")
			f = strings.TrimSuffix(f, "-Italic")
			words = append(words,
				pdf.Text{Font: f, FontSize: ck.FontSize, X: ck.X, Y: ck.Y, W: end - ck.X, S: s})
			k = l
		}
		i = j
	}

	return words
}

func sameFont(f1, f2 string) bool {
	f1 = strings.TrimSuffix(f1, ",Italic")
	f1 = strings.TrimSuffix(f1, "-Italic")
	f2 = strings.TrimSuffix(f1, ",Italic")
	f2 = strings.TrimSuffix(f1, "-Italic")
	return strings.TrimSuffix(f1, ",Italic") == strings.TrimSuffix(f2, ",Italic") || f1 == "Symbol" || f2 == "Symbol" || f1 == "TimesNewRoman" || f2 == "TimesNewRoman"
}

func registerHandler(aHandler handler) {
	h = aHandler
}
