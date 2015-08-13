package main

import (
	"fmt"
	"log"
	"os"

	"mcesar.io/ofx"
)

func main() {
	doc, err := ofx.Parse(os.Stdin)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	for _, t := range doc.Transactions {
		fmt.Printf("\"%v:%v\",%v,\"%v\"\n", "<*>", -t.Amount, t.Date, t.Memo)
	}
}
