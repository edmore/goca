// Go(lang) Matterhorn CA
package main

import (
	"encoding/json"
	"fmt"
	"github.com/edmore/goca/auth"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	fmt.Println(config.AdminServerURL)

	client := &http.Client{}
	name, _ := os.Hostname()

	// Register CA
	req, err := http.NewRequest(
		"POST",
		config.AdminServerURL+"/capture-admin/agents/"+config.Name+"?address=http://"+name+":8080&state=idle",
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
	fmt.Println("Request: ", req)
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	// Print Schedule
}
