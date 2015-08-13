package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"mcesar.io/ofx"
)

type substitution struct {
	pattern string
	account string
	memo    string
}

func main() {
	substitutionsFile := flag.String("s", "", "substitutions file")
	flag.Parse()

	substitutions := []substitution{}
	if *substitutionsFile != "" {
		f, err := os.Open(*substitutionsFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		csvReader := csv.NewReader(f)
		for {
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			} else if err != nil && err != io.EOF {
				fmt.Fprintln(os.Stderr, "reading substitutions file:", err)
				return
			}
			substitutions = append(substitutions, substitution{record[0], record[1], record[2]})
		}
	}

	doc, err := ofx.Parse(os.Stdin)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	st := strings.Title
	sl := strings.ToLower
	for _, t := range doc.Transactions {
		account := "<*>"
		t.Memo = st(sl(t.Memo))
		for _, s := range substitutions {
			if matched, err := regexp.MatchString(sl(s.pattern), sl(t.Memo)); matched && err == nil {
				account = s.account
				t.Memo = s.memo
				break
			}
		}
		fmt.Printf("\"%v:%v\",%v,\"%v\"\n", account, -t.Amount, t.Date, t.Memo)
	}
}
