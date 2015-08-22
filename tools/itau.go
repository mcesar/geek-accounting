package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"rsc.io/pdf"
)

type itau struct {
	started bool
	date    string
	memo    string
	year    string
	ch      chan transaction
}

func init() {
	registerHandler(&itau{})
}

func (i *itau) start(ch chan transaction) {
	i.started = false
	i.ch = ch
}

func (i *itau) handleWord(t *pdf.Text, page int) bool {
	if t.X == 207.6 && t.S == "Saldo anterior" {
		i.started = true
		return false
	}
	if i.started && t.X == 209.52 && t.S == "Saldo em C/C" {
		return true
	}
	if i.started && t.X >= 195 && t.X <= 210 &&
		t.S != "descrição" && t.S != "(-) Saldo a liberar" &&
		t.S != "Saldo final disponivel" {
		if i.memo != "" {
			fmt.Fprintf(os.Stderr, "Malformed: %v\n", t.S)
			os.Exit(2)
		}
		i.memo = t.S
	}
	if i.started && t.X >= 150 && t.X <= 153 && t.S != "data" {
		arr := strings.Split(t.S, "/")
		i.date = i.year + "-" + arr[1] + "-" + arr[0]
	}
	if i.started && (t.X+t.W >= 457 && t.X+t.W <= 461 || t.X+t.W >= 394 && t.X+t.W <= 398) &&
		t.S != "saídas R$" && t.S != "entradas R$" &&
		t.S != "(débitos)" && t.S != "(créditos)" {
		var (
			amount float64
			err    error
		)
		t.S = strings.Replace(t.S, ".", "", -1)
		t.S = strings.Replace(t.S, ",", ".", 1)
		if strings.HasSuffix(t.S, "-") {
			amount, err = strconv.ParseFloat("-"+t.S[0:len(t.S)-1], 64)
		} else {
			amount, err = strconv.ParseFloat(t.S, 64)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing amount: %v", t.S)
			os.Exit(2)
		}
		i.ch <- transaction{i.date, amount, i.memo}
		i.memo = ""
	}
	if !i.started && page == 1 && t.X >= 514 && t.X <= 515 && t.Y >= 763 && t.Y <= 764 {
		i.year = t.S[4:]
	}
	return false
}
