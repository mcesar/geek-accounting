package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"google.golang.org/api/drive/v2"

	"github.com/mcesar/geek-accounting/tools/gdrive"
)

func main() {
	folderId := flag.String("f", "", "Folder Id")
	secretFile := flag.String("s", "", "Secret file")
	destination := flag.String("d", "", "Destination")
	flag.Parse()

	client, err := gdrive.GetClient(*secretFile)
	if err != nil {
		log.Fatalf("Unable to retrieve Client %v", err)
	}

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}

	if *destination == "" {
		*destination, err = ioutil.TempDir("", "ga-tools")
		if err != nil {
			log.Fatal(err)
		}
	}

	dowloadFilesInFolder(client, srv, *folderId, *destination)
}

func dowloadFilesInFolder(client *http.Client, srv *drive.Service, folderId, destination string) {
	files, err := gdrive.FilesInFolder(srv, folderId)
	if err != nil {
		log.Fatalf("Unable to retrieve files.", err)
	}
	for _, f := range files {
		if (f.MimeType == "application/pdf" ||
			f.MimeType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet") &&
			f.DownloadUrl != "" {
			if err := gdrive.DownloadFile(client, f, destination); err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("%s\n", filepath.Join(destination, f.Title))
		} else if f.MimeType == "application/vnd.google-apps.folder" {
			os.Mkdir(destination+"/"+f.Title, os.ModeDir)
			dowloadFilesInFolder(client, srv, f.Id, destination+"/"+f.Title)
		}
	}
}
