package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mcesar/geek-accounting/tools/gdrive"
	"google.golang.org/api/drive/v2"
)

func main() {
	inputFile := flag.String("f", "", "Input file Id")
	destination := flag.String("d", "", "Destination folder Id")
	newFolder := flag.String("n", "", "New folder name")
	secretFile := flag.String("s", "", "Secret file")
	flag.Parse()

	client, err := gdrive.GetClient(*secretFile)
	if err != nil {
		log.Fatalf("Unable to obtain client: %v", err)
	}

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}

	var f *os.File
	if *inputFile == "" {
		f = os.Stdin
	} else {
		f, err = os.Open(*inputFile)
		if err != nil {
			log.Fatalf("Unable to open file", *inputFile)
		}
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		arr := strings.Split(s.Text(), "/")
		n := strings.Split(arr[len(arr)-1], ".pdf")[0]
		n = strings.Split(n, ".xlsx")[0]
		for _, r := range [][]string{
			{"Ã", "Ã"}, {"Ç", "Ç"}, {"Ê", "Ê"}, {"Á", "Á"}, {"É", "É"}, {"á", "á"},
			{"ç", "ç"}, {"ã", "ã"}, {"ê", "ê"}, {"é", "é"}} {
			n = strings.Replace(n, r[0], r[1], -1)
		}
		files, err := gdrive.FilesWithTitle(srv, n)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to retrieve file. %v\n", err)
			return
		}
		fileId := ""
		retrieved := make([]string, 0, len(files))
		for _, f := range files {
			if strings.HasPrefix(f.Title, n+".pdf") || strings.HasPrefix(f.Title, n+".xlsx") {
				if fileId == "" {
					fileId = f.Id
				}
				retrieved = append(retrieved, f.Title)
			}
		}
		if len(retrieved) != 1 {
			fmt.Fprintf(os.Stderr, "Unable to locate file. %v %v\n", n, retrieved)
		} else {
			var folderid string
			if *newFolder == "" {
				folderid = *destination
			} else {
				files, err = gdrive.FilesWithTitle(srv, *newFolder)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Unable to retrieve file. %v\n", err)
					return
				}
				for _, f := range files {
					for _, p := range f.Parents {
						if p.Id == *destination {
							folderid = f.Id
						}
					}
				}
				if folderid == "" {
					folderid, err = gdrive.CreateFolder(srv, *newFolder, *destination)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Unable to create folder %v.\n", *newFolder)
						return
					}
				}
			}
			if err := gdrive.MoveFile(srv, fileId, folderid); err != nil {
				fmt.Fprintf(os.Stderr, "Unable to move file %v: %v.\n", n, err)
			}
		}
	}
	if err := s.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input file:", err)
	}
}
