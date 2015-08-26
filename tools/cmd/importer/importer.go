package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	pkg := os.Getenv("PKG")
	if pkg == "" {
		log.Fatal("Please set $PKG variable")
	}
	f, err := os.Create("import.go")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf(`package main
	
import _ "%v"
`, pkg))

}
