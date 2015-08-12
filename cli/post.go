package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {

	if len(os.Args) < 4 {
		printUsage()
		return
	}

	var url = os.Args[1]
	var coa = os.Args[2]
	var account = os.Args[3]

	if url == "" || coa == "" || account == "" {
		printUsage()
		return
	}

	var buf bytes.Buffer

	csvReader := csv.NewReader(os.Stdin)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			return
		} else {
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
			if sum < 0 {
				debits = append(debits,
					fmt.Sprintf("{\"account\":\"%v\", \"value\":%v}", account, -sum))
			} else {
				credits = append(credits,
					fmt.Sprintf("{\"account\":\"%v\", \"value\":%v}", account, sum))
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
		}
	}
	fmt.Println(string(buf.Bytes()))
	req, err := http.NewRequest("POST", url+"/charts-of-accounts/"+coa+"/transactions", &buf)
	if err != nil {
		panic(err)
	}
	req.Close = false
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", "admin")

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
	fmt.Println("usage: post url coa account")
}
