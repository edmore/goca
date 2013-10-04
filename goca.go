// Go(lang) Matterhorn CA
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/edmore/goca/auth"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	//"reflect"
)

type Config struct {
	AdminServerURL string // the admin node URL
	Name           string // the Capture Agent Name
	DigestUser     string // Digest User
	DigestPassword string // Digest Password
}

var config *Config

func loadConfig() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Println("open config: ", err)
	}

	temp := new(Config)
	if err = json.Unmarshal(file, temp); err != nil {
		log.Println("parse config: ", err)
	}
	config = temp
}

func main() {
	// Load config
	loadConfig()

	client := &http.Client{}
	hostName, _ := os.Hostname()

	// Register CA
	req, err := http.NewRequest(
		"POST",
		config.AdminServerURL+"/capture-admin/agents/"+config.Name+"?address=http://"+hostName+":8080&state=idle",
		nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("X-Requested-Auth", "Digest")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	req = auth.SetDigestAuth(req, config.DigestUser, config.DigestPassword, resp, 1)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	// Get Schedule
	req, err = http.NewRequest(
		"GET",
		config.AdminServerURL+"/recordings/calendars?agentid="+config.Name,
		nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("X-Requested-Auth", "Digest")
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	req = auth.SetDigestAuth(req, config.DigestUser, config.DigestPassword, resp, 1)
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	// Defer the closing of the body
	defer resp.Body.Close()
	// Read the content into a byte array
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	vcal := string(body)

	// Read the cal line by line
	cal := strings.NewReader(vcal)
	scanner := bufio.NewScanner(cal)
	for scanner.Scan() {
		r, _ := regexp.Compile("VEVENT")
		if r.Match([]byte(scanner.Text())) {
			fmt.Println(scanner.Text())
		}
	}
}
