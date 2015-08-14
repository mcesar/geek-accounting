package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"rsc.io/pdf"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pdf2csv file.pdf")
		os.Exit(2)
	}
	r, err := pdf.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	var year string
	start := false
PageLoop:
	for i := 1; i <= r.NumPage(); i++ {
		words := findWords(r.Page(i).Content().Text)
		for _, t := range words {
			if t.X == 207.6 && t.S == "Saldo anterior" {
				start = true
				continue
			}
			if start && t.X == 209.52 && t.S == "Saldo em C/C" {
				break PageLoop
			}
			if start && t.X >= 195 && t.X <= 210 && t.S != "descrição" {
				fmt.Println(t.S)
			}
			if start && t.X >= 150 && t.X <= 153 && t.S != "data" {
				fmt.Println(t.S)
			}
			if start && t.X+t.W >= 457 && t.X+t.W <= 461 &&
				t.S != "saídas R$" && t.S != "(débitos)" {
				fmt.Println(t.S)
			}
			if start && t.X+t.W >= 394 && t.X+t.W <= 396 &&
				t.S != "saídas R$" && t.S != "(débitos)" {
				fmt.Println(t.S)
			}
			if !start && i == 1 && t.X >= 514 && t.X <= 515 && t.Y >= 763 && t.Y <= 764 {
				year = t.S[4:]
			}
		}
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
			words = append(words, pdf.Text{f, ck.FontSize, ck.X, ck.Y, end - ck.X, s})
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

// {MyriadPro-SemiCn 7.67 207.6 230.72 41.26459999999997 Saldo anterior}
// {MyriadPro-SemiCn 7.67 207.6 206.96 80.94151000000008 Rshop-CLAERRO SOR-02/01}
// {MyriadPro-SemiCn 7.67 200.88 197.12 69.75973000000005 P Remuneração/Salário}
// {MyriadPro-SemiCn 7.67 209.52 723.44 72.67325000000002 Rshop-LE BIJOUX -13/01}
// {MyriadPro-SemiCn 7.67 150.48 206.96 17.05807999999999 02/02}
// {MyriadPro-SemiCn 7.67 439.92 206.96 18.41567000000009 11,00-}
// {MyriadPro-SemiCn 7.67 430.56 51.2 27.25918000000013 2.178,64-} 458.336
