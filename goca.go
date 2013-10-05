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
	//	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AdminServerURL string // the admin node URL
	Name           string // the Capture Agent Name
	DigestUser     string // Digest User
	DigestPassword string // Digest Password
}

type Event struct {
	Dtstamp  string
	Dtstart  string
	Dtend    string
	Summary  string
	Uid      int
	Location string
}

var config *Config
var events []*Event

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
	// [Load config]
	loadConfig()

	client := &http.Client{}
	hostName, _ := os.Hostname()

	// [Register CA]
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

	// [Get Schedule]
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

	// Read the ical line by line
	cal := strings.NewReader(vcal)
	scanner := bufio.NewScanner(cal)
	for scanner.Scan() {
		begin, _ := regexp.Compile("^BEGIN:VEVENT")
		dtstamp, _ := regexp.Compile("^DTSTAMP")
		dtstart, _ := regexp.Compile("^DTSTART")
		dtend, _ := regexp.Compile("^DTEND")
		summary, _ := regexp.Compile("^SUMMARY")
		uid, _ := regexp.Compile("^UID")
		location, _ := regexp.Compile("^LOCATION")
		end, _ := regexp.Compile("^END:VEVENT")

		if begin.Match([]byte(scanner.Text())) {
			e := new(Event)
			scanner.Scan()
			for !end.Match([]byte(scanner.Text())) {
				current := []byte(scanner.Text())
				switch {
				case dtstamp.Match(current):
					e.Dtstamp = strings.Split(scanner.Text(), ":")[1]
				case dtstart.Match(current):
					e.Dtstart = strings.Split(scanner.Text(), ":")[1]
				case dtend.Match(current):
					e.Dtend = strings.Split(scanner.Text(), ":")[1]
				case summary.Match(current):
					e.Summary = strings.Split(scanner.Text(), ":")[1]
				case uid.Match(current):
					e.Uid, _ = strconv.Atoi(strings.Split(scanner.Text(), ":")[1])
				case location.Match(current):
					e.Location = strings.Split(scanner.Text(), ":")[1]
				}
				scanner.Scan()
			}
			currentTime := time.Now().UTC()
			fmt.Println(currentTime)
			events = append(events, e)
		}
	}
	// Print the events
	for _, v := range events {
		fmt.Println(v)
	}
}
