package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
)

type SystemCommandResponse struct {
	Command 	string
	StdOut 		string
	StdErr 		string
}

type SystemCommand struct {
	Command string
}

func (app *webSocketApp) handleSystemCommand(w http.ResponseWriter, r *http.Request) {
	var systemCommand SystemCommand
	err := json.NewDecoder(r.Body).Decode(&systemCommand)
	if err != nil {
		log.Println("Error decoding request ", err)
		w.WriteHeader(500)
		return
	}	
	
	log.Println("Command given: " + systemCommand.Command)

	cmd := exec.Command("bash", "-c", systemCommand.Command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Println(cmdErr)
	}
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	log.Println("OutStr: " + outStr + ", ErrStr: " + errStr)
	var commandResult = SystemCommandResponse {
		Command:	systemCommand.Command,
		StdOut:		outStr,
		StdErr:		errStr,
	}
	
	js, err := json.Marshal(commandResult)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}