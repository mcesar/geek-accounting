package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"mcesar.io/ofx"
)

type substitution struct {
	pattern  string
	account  string
	memo     string
	minValue float64
	maxValue float64
}

func main() {
	substitutionsFile := flag.String("s", "", "substitutions file")
	filename := flag.String("f", "", "file name")
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
			s := substitution{record[0], record[1], record[2], 0, 0}
			if len(record) > 3 {
				if s.minValue, err = strconv.ParseFloat(record[3], 64); err != nil {
					fmt.Fprintln(os.Stderr, "Error converting minValue:", err)
				}
			}
			if len(record) > 4 {
				if s.maxValue, err = strconv.ParseFloat(record[4], 64); err != nil {
					fmt.Fprintln(os.Stderr, "Error converting maxValue:", err)
				}
			}
			substitutions = append(substitutions, s)
		}
	}
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
	st := strings.Title
	sl := strings.ToLower
	for _, t := range doc.Transactions {
		account := "<*>"
		t.Memo = st(sl(t.Memo))
		for _, s := range substitutions {
			if matched, err := regexp.MatchString(sl(s.pattern), sl(t.Memo)); matched && err == nil {
				if s.minValue > 0 && math.Abs(t.Amount) < s.minValue {
					continue
				}
				if s.maxValue > 0 && math.Abs(t.Amount) > s.maxValue {
					continue
				}
				account = s.account
				t.Memo = s.memo
				break
			}
		}
		fmt.Printf("\"%v:%v\",%v,\"%v\"\n", account, -t.Amount, t.Date, t.Memo)
	}
}
