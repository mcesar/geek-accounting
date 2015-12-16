// +build ignore

package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {

	account := flag.String("a", "", "account")
	username := flag.String("u", "admin", "user name")
	filename := flag.String("f", "", "file name")
	flag.Parse()

	url := flag.Arg(0)
	coa := flag.Arg(1)

	if flag.NArg() < 2 {
		printUsage()
		return
	}

	if url == "" || coa == "" {
		printUsage()
		return
	}

	var (
		buf bytes.Buffer
		f   *os.File
	)
	if *filename == "" {
		f = os.Stdin
	} else {
		var err error
		if f, err = os.Open(*filename); err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file:", err)
			return
		}
	}

	count := 0
	fmt.Fprintf(&buf, `{ "__multi__": true }`)
	csvReader := csv.NewReader(f)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, "Error reading standard input:", err)
			return
		}
		debits := []string{}
		credits := []string{}
		sum := 0.0
		for _, e := range strings.Split(record[0], ",") {
			earr := strings.Split(e, ":")
			f, err := strconv.ParseFloat(earr[1], 64)
			if err != nil {
				panic(err)
			}
			sum += f
			if strings.HasPrefix(earr[1], "-") {
				credits = append(credits,
					fmt.Sprintf("{\"account\":\"%v\", \"value\":%v}", earr[0], earr[1][1:]))
			} else {
				debits = append(debits,
					fmt.Sprintf("{\"account\":\"%v\", \"value\":%v}", earr[0], earr[1]))
			}
		}
		if *account == "" {
			if int64(sum*100) != 0 {
				fmt.Fprintf(os.Stderr, "Debits plus credits is different from 0: %v %v\n",
					sum, record[2])
				return
			}
		} else {
			if sum < 0 {
				debits = append(debits,
					fmt.Sprintf("{\"account\":\"%v\", \"value\":%v}", *account, -sum))
			} else if sum > 0 {
				credits = append(credits,
					fmt.Sprintf("{\"account\":\"%v\", \"value\":%v}", *account, sum))
			}
		}
		fmt.Fprintf(&buf, `{ "debits": [`)
		for i, d := range debits {
			if i > 0 {
				fmt.Fprintf(&buf, ", ")
			}
			fmt.Fprintf(&buf, d)
		}
		fmt.Fprintf(&buf, `], "credits": [`)
		for i, d := range credits {
			if i > 0 {
				fmt.Fprintf(&buf, ", ")
			}
			fmt.Fprintf(&buf, d)
		}
		fmt.Fprintf(&buf, `], "date": "%vT00:00:00Z", "memo": "%v" }`, record[1], record[2])
		count++
	}
	fmt.Fprintf(&buf, `{ "__count__": %v}`, count)
	oldState, err := terminal.MakeRaw(1)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(1, oldState)
	fmt.Print("Password: ")
	pw, err := terminal.ReadPassword(1)
	if err != nil {
		panic(err)
	}
	password := string(pw)

	req, err := http.NewRequest("POST", url+"/charts-of-accounts/"+coa+"/transactions", &buf)
	if err != nil {
		panic(err)
	}
	req.Close = false
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(*username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

func printUsage() {
	fmt.Println("usage: post url coa [-a account] [-u user] [-f filename]")
}
