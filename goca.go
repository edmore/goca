// Go(lang) Matterhorn CA
package main

import (
	"bufio"
	"encoding/json"
	"github.com/edmore/goca/auth"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	//	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AdminServerURL string // the admin node URL
	Name           string // the Capture Agent Name
	DigestUser     string // Digest User
	DigestPassword string // Digest Password
	CaptureDir     string // Directory for captures
}

type Event struct {
	Dtstamp  time.Time
	Dtstart  time.Time
	Dtend    time.Time
	Summary  string
	Uid      int
	Location string
}

type Events []*Event

// for sorting
func (s Events) Len() int      { return len(s) }
func (s Events) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// ByDtstart implements sort.Interface by providing Less and using the Len and
// Swap methods of the embedded Events value.
type ByDtstart struct{ Events }

func (s ByDtstart) Less(i, j int) bool {
	return s.Events[i].Dtstart.Before(s.Events[j].Dtstart)
}

var config *Config

const updateFrequency time.Duration = 60 * time.Second

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

func strToTime(s string) time.Time {
	layout := "20060102T150405Z"
	t, _ := time.Parse(layout, s)
	return t
}

func getTimeStamp() time.Time {
	t := time.Now().UTC()
	return t
}

func registerCA(state string) {
	client := &http.Client{}
	hostName, _ := os.Hostname()
	req, err := http.NewRequest(
		"POST",
		config.AdminServerURL+"/capture-admin/agents/"+config.Name+"?address=http://"+hostName+":8080&state="+state,
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
}

func getSchedule() {
	var events Events
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		config.AdminServerURL+"/recordings/calendars?agentid="+config.Name,
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
					e.Dtstamp = strToTime(strings.Split(scanner.Text(), ":")[1])
				case dtstart.Match(current):
					e.Dtstart = strToTime(strings.Split(scanner.Text(), ":")[1])
				case dtend.Match(current):
					e.Dtend = strToTime(strings.Split(scanner.Text(), ":")[1])
				case summary.Match(current):
					e.Summary = strings.Split(scanner.Text(), ":")[1]
				case uid.Match(current):
					e.Uid, _ = strconv.Atoi(strings.Split(scanner.Text(), ":")[1])
				case location.Match(current):
					e.Location = strings.Split(scanner.Text(), ":")[1]
				}
				scanner.Scan()
			}
			// if the event has not passed
			if e.Dtend.After(e.Dtstamp) {
				events = append(events, e)
			}
		}
	}
	// sort the events
	sort.Sort(ByDtstart{events})
	ch <- events
}

func recordingState(recordingId, state string) {
	client := &http.Client{}
	req, err := http.NewRequest(
		"POST",
		config.AdminServerURL+"/capture-admin/recordings/"+recordingId+"?state="+state,
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
}

func startCapture(e *Event) {
	now := getTimeStamp()
	log.Println("Starting capture ", e.Uid)
	duration := e.Dtend.Sub(now)
	log.Println("Duration set to ", duration)
	recordingId := strconv.Itoa(e.Uid)
	recordingName := "recording-" + recordingId
	recordingDir := config.CaptureDir + "/" + recordingName
	os.MkdirAll(recordingDir, 0755)

	// Set CA state
	state = "capturing"
	go registerCA(state)
	// Set recording status
	go recordingState(recordingId, "capturing")

	// TODO : record from streams
}

var (
	ch    = make(chan Events)
	state string
)

func main() {
	state = "idle"
	var scheduled Events
	var lastUpdate time.Time = getTimeStamp()
	// [Load config]
	loadConfig()
	// [Register CA]
	registerCA(state)
	// [Get Schedule]
	go getSchedule()
	// [Control Loop]
	for {
		select {
		case events := <-ch:
			// Update and Print the updated schedule
			scheduled = events
			log.Println("Schedule update successful.")
			// TODO : Get the start of capture just right
		case <-time.After(2 * time.Second):
			now := getTimeStamp()
			if now.After(scheduled[0].Dtstart) || now.Equal(scheduled[0].Dtstart) {
				if state == "idle" {
					go startCapture(scheduled[0])
				}
			}
			if now.Sub(lastUpdate) > updateFrequency {
				go getSchedule()
				lastUpdate = getTimeStamp()
			}
		}
	}
}
