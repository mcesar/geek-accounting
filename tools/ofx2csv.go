package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"mcesar.io/ofx"
)

func main() {
	substitutionsFile := flag.String("s", "", "substitutions file")
	filename := flag.String("f", "", "file name")
	flag.Parse()

	var f *os.File
	if *filename == "" {
		f = os.Stdin
	} else {
		var err error
		if f, err = os.Open(*filename); err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file:", err)
			return
		}
	}

	doc, err := ofx.Parse(f)
	if err != nil {
		log.Fatal("Error parsing OFX: ", err)
		os.Exit(1)
	}
	var substitutions Substitutions
	if *substitutionsFile != "" {
		if substitutions, err = NewSubstitutions(*substitutionsFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
	for _, t := range doc.Transactions {
		account, amount, memo := substitutions.ReplaceData(t.Amount, t.Memo)
		fmt.Printf("\"%v:%v\",%v,\"%v\"\n", account, amount, t.Date, memo)
	}
}
