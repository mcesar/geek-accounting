package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Substitutions []substitution

type substitution struct {
	pattern  string
	account  string
	memo     string
	minValue float64
	maxValue float64
}

func NewSubstitutions(file string) (Substitutions, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	substitutions := Substitutions{}
	csvReader := csv.NewReader(f)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil && err != io.EOF {
			return nil, fmt.Errorf("reading substitutions file: %v", err)
		}
		s := substitution{record[0], record[1], record[2], 0, 0}
		if len(record) > 3 {
			if s.minValue, err = strconv.ParseFloat(record[3], 64); err != nil {
				return nil, fmt.Errorf("Error converting minValue: %v", err)
			}
		}
		if len(record) > 4 {
			if s.maxValue, err = strconv.ParseFloat(record[4], 64); err != nil {
				return nil, fmt.Errorf("Error converting maxValue: %v", err)
			}
		}
		substitutions = append(substitutions, s)
	}
	return substitutions, nil
}

func (ss Substitutions) ReplaceData(amount float64, memo string) (string, float64, string) {
	st := strings.Title
	sl := strings.ToLower
	account := "<*>"
	memo = st(sl(memo))
	for _, s := range ss {
		if matched, err := regexp.MatchString(sl(s.pattern), sl(memo)); matched && err == nil {
			if s.minValue > 0 && math.Abs(amount) < s.minValue {
				continue
			}
			if s.maxValue > 0 && math.Abs(amount) > s.maxValue {
				continue
			}
			account = s.account
			memo = s.memo
			break
		}
	}
	return account, -amount, memo
}
