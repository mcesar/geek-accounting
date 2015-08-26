//go:generate importer

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	toolsPdf "github.com/mcesarhm/geek-accounting/tools/pdf"
	"github.com/mcesarhm/geek-accounting/tools/substitutions"

	"rsc.io/pdf"
)

func main() {
	substitutionsFile := flag.String("s", "", "substitutions file")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "usage: pdf2csv file.pdf [-s substitutions.csv]")
		os.Exit(2)
	}
	r, err := pdf.Open(flag.Arg(0))
	if err != nil {
		log.Fatal("Error opening pdf: ", err)
	}
	var s substitutions.Substitutions
	if *substitutionsFile != "" {
		if s, err = substitutions.NewSubstitutions(*substitutionsFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
	ch := make(chan toolsPdf.Transaction)
	go func() {
	PageLoop:
		for i := 1; i <= r.NumPage(); i++ {
			words := toolsPdf.FindWords(r.Page(i).Content().Text)
			for _, t := range words {
				if toolsPdf.HandleWord(&t, i, ch) {
					break PageLoop
				}
			}
		}
		close(ch)
	}()
	for t := range ch {
		account, amount, memo := s.ReplaceData(t.Amount, t.Memo)
		fmt.Printf("\"%v:%v\",%v,\"%v\"\n", account, amount, t.Date, memo)
	}
}
