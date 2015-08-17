// +build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tealeg/xlsx"
)

func main() {
	f, err := xlsx.OpenFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	for _, sheet := range f.Sheets {
		for _, row := range sheet.Rows {
			for _, cell := range row.Cells {
				fmt.Printf("%s\n", cell.String())
			}
		}
	}
}
