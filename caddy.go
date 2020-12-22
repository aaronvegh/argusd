package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"net/http"
	"strings"
)

type CaddyConfigResponse struct {
	Name string
	Config string
}

func handleGetCaddyConfig(w http.ResponseWriter, r *http.Request) {
	
	argusDirectory := "/etc/caddy/Caddyfile-Argus.d"
	
	caddyExists, _ := exists("/usr/bin/caddy")
	if !caddyExists {
		log.Println("Caddy not installed!")
		w.WriteHeader(503) // service not available, dawg!
		return
	}
	log.Println("Caddy is installed.")
	
	dirExists, _ := exists(argusDirectory)
	if !dirExists {
		err := os.Mkdir(argusDirectory, 0755)
		if err != nil {
			log.Println("Error creating argus caddy directory ", err)
			w.WriteHeader(500)
			return
		}
	}
	log.Println("Argus Caddy Config Directory exists.")
	
	// confirm configuration is setup for our config directory
	cmd := exec.Command("bash", "-c", "grep -Fxq \"import Caddyfile-Argus.d/*.caddyfile\" /etc/caddy/Caddyfile; echo $?")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Println("Failed to run check on configuration: ", cmdErr)
		w.WriteHeader(500)
		return
	}
	log.Println("Successfully executed Config check. Result is ", out.String())
	
	if strings.Contains(out.String(), "1") {
		// replace default caddyfile with ours
		mvErr := os.Rename("/etc/caddy/Caddyfile", "/etc/caddy/Caddyfile-orig")
		if mvErr != nil {
			log.Println("Couldn't replace Caddyfile: ", mvErr)
			w.WriteHeader(500)
			return
		}
		log.Println("Moved original Caddyfile to backup.")
		
		// inject import line to Caddyfile
		inject := exec.Command("bash", "-c", "echo \"import Caddyfile-Argus.d/*.caddyfile\" >> /etc/caddy/Caddyfile")
		var injectOut bytes.Buffer
		cmd.Stdout = &injectOut
		injectErr := inject.Run()
		if injectErr != nil {
			log.Println("Coudn't inject Argus Caddyfile directive: ", injectErr)
			w.WriteHeader(500)
			return
		}
		log.Println("Created new Caddyfile.")
	}
	
	
	directoryContents, err := ioutil.ReadDir(argusDirectory)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	log.Println("Read contents of Argus Caddy directory.")
	
	var caddyConfigResponse []CaddyConfigResponse
	for _, info := range directoryContents {
		filePath := argusDirectory + "/" + info.Name()
		fileContents, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		
		caddyConfigResponse = append(caddyConfigResponse, CaddyConfigResponse {
			Name: 	info.Name(),
			Config: string(fileContents),
		})
	}
	
	js, err := json.Marshal(caddyConfigResponse)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func handleSetCaddyConfig(w http.ResponseWriter, r *http.Request) {
	
	argusDirectory := "/etc/caddy/Caddyfile-Argus.d"
	
	var caddyConfig []CaddyConfigResponse
	err := json.NewDecoder(r.Body).Decode(&caddyConfig)
	if err != nil {
		log.Println("Error decoding request ", err)
		w.WriteHeader(500)
		return
	}
	
	// Delete the contents of this directory so we can rewrite it
	directoryContents, err := ioutil.ReadDir(argusDirectory)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	
	for _, info := range directoryContents {
		filePath := argusDirectory + "/" + info.Name()
		err := os.Remove(filePath)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
	}
	
	for _, configuration := range caddyConfig {
		configDestination := argusDirectory + "/" + configuration.Name
		configExists, _ := exists(configDestination)
		var configFile *os.File
		if !configExists {
			configFile, err = os.Create(configDestination)
			if err != nil {
				log.Println("Error creating config file ", err)
				w.WriteHeader(500)
				return
			}
		} else {
			configFile, err = os.Open(configDestination)
			if err != nil {
				log.Println("Error opening existing config file ", err)
				w.WriteHeader(500)
				return
			}
		}
		
		_, err := configFile.Write([]byte(configuration.Config))
		if err != nil {
			log.Println("Error writing new config to file ", err)
			w.WriteHeader(500)
			return
		}
	}
	
	cmd := exec.Command("bash", "-c", "systemctl reload caddy")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Println(cmdErr)
	}
	
	w.WriteHeader(200)
}