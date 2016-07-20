package gdrive

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func GetClient(secretFile string) (*http.Client, error) {
	config, err := oauthConfig(secretFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to obtain oauth config: %v", err)
	}
	cacheFile, err := tokenCacheFile()
	if err != nil {
		return nil, fmt.Errorf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	ctx := context.Background()
	return config.Client(ctx, tok), nil
}

func oauthConfig(secretFile string) (*oauth2.Config, error) {
	b, err := ioutil.ReadFile(secretFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to read client secret file: %v", err)
	}
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse client secret file to config: %v", err)
	}
	return config, nil
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("drive-api-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func FilesWithTitle(d *drive.Service, title string) ([]*drive.File, error) {
	return filesWithQuery(d, fmt.Sprintf("title contains '%v'", title))
}

func FilesInFolder(d *drive.Service, folderId string) ([]*drive.File, error) {
	return filesWithQuery(d, fmt.Sprintf("'%v' in parents", folderId))
}

func filesWithQuery(d *drive.Service, query string) ([]*drive.File, error) {
	var fs []*drive.File
	q := d.Files.List().Q(query)
	pageToken := ""
	for {
		// If we have a pageToken set, apply it to the query
		if pageToken != "" {
			q = q.PageToken(pageToken)
		}
		r, err := q.Do()
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
			return fs, err
		}
		fs = append(fs, r.Items...)
		pageToken = r.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return fs, nil
}

func DownloadFile(client *http.Client, f *drive.File, folder string) error {
	res, err := client.Get(f.DownloadUrl)
	if err != nil {
		return fmt.Errorf("An error occurred: %v", err)
	}
	defer res.Body.Close()

	outFile, err := os.Create(filepath.Join(folder, f.Title))
	if err != nil {
		return fmt.Errorf("An error occurred: %v\n", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, res.Body)
	if err != nil {
		return fmt.Errorf("An error occurred: %s", err)
	}
	return nil
}

func MoveFile(d *drive.Service, fileId, folderId string) error {
	r, err := d.Parents.List(fileId).Do()
	if err != nil {
		return err
	}
	if len(r.Items) == 1 && r.Items[0].Id == folderId {
		return nil
	}
	p := &drive.ParentReference{Id: folderId}
	_, err = d.Parents.Insert(fileId, p).Do()
	if err != nil {
		return err
	}
	for _, p := range r.Items {
		err := d.Parents.Delete(fileId, p.Id).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateFolder(d *drive.Service, folderName, parentId string) (string, error) {
	f := &drive.File{Title: folderName, MimeType: "application/vnd.google-apps.folder"}
	p := &drive.ParentReference{Id: parentId}
	f.Parents = []*drive.ParentReference{p}
	r, err := d.Files.Insert(f).Do()
	if err != nil {
		return "", err
	}
	return r.Id, nil
}
