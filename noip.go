/*
 * Copyright (c) 2015 Pier Luigi Fiorini
 *
 * Permission is hereby granted, free of charge, to any person obtaining a
 * copy of this software and associated documentation files (the "Software"),
 * to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
 * DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
 * OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE
 * USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"strings"
)

// Flags
var (
	hostnameFlag = flag.String("hostname", "", "the hostname to update")
	ipFlag       = flag.String("ip", "", "use this ip address instead of detect it")
)

// Read authentication data from ~/.netrc.
func readAuthData() (string, string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", "", err
	}

	file, err := os.Open(usr.HomeDir + "/.netrc")
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	username := ""
	password := ""

	defaultRe := regexp.MustCompile(`^default\s+login (.+)\s+password (.+)`)
	machineRe := regexp.MustCompile(`^machine (.+)\s+login (.+)\s+password (.+)`)

	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if defaultRe.MatchString(line) {
			submatches := machineRe.FindStringSubmatch(line)
			if len(submatches) == 3 {
				// Default takes precedence over anything else
				username = submatches[1]
				password = submatches[2]
				return username, password, nil
			}
		}
		if machineRe.MatchString(line) {
			submatches := machineRe.FindStringSubmatch(line)
			if len(submatches) == 4 && submatches[1] == "dynupdate.no-ip.com" {
				// Save the last authentication for the address we are interested in
				username = submatches[2]
				password = submatches[3]
			}
		}
	}

	return username, password, nil
}

// Determine external IP address.
func getExternalIp() (string, error) {
	res, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	buffer, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(buffer), "\n"), nil
}

// Update IP address for a host name.
func updateIp(hostname string, ip string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://dynupdate.no-ip.com/nic/update?hostname=%s&myip=%s", hostname, ip), nil)
	if err != nil {
		return "", err
	}
	username, password, err := readAuthData()
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(username, password)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	buffer, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(buffer), "\n"), nil
}

func main() {
	flag.Parse()

	if *hostnameFlag == "" {
		log.Fatalln("Must specify the hostname to update")
	}

	var ip string = *ipFlag
	if ip == "" {
		var err error
		ip, err = getExternalIp()
		if err != nil {
			log.Fatalln(err)
		}
	}
	if ip == "" {
		log.Fatalln("Unable to determine the IP address, aborting...")
	}

	resp, err := updateIp(*hostnameFlag, ip)
	if err != nil {
		log.Fatalln(err)
	}

	if !strings.HasPrefix(resp, "good") && !strings.HasPrefix(resp, "nochg") {
		fmt.Printf("Error: %s\n", resp)
		os.Exit(1)
	}
}
