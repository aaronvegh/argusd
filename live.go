package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type LiveRequest struct {
	RequestName string
	RequestBody interface{}
}

type InstallRequest struct {
	Distro   string
	Packages string
}

type LogFileRequest struct {
	Command string 	// tail or cat or zcat...
	Path string		// the logfile path
	Options string	// things like "-f" or "-n 50"
}

func (app *webSocketApp) handleLiveResponse(w http.ResponseWriter, r *http.Request) {
	log.Println("This is /liveResponse starting")

	conn, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	session := &webSocketSession{
		connection: conn,
	}

	for {
		var body json.RawMessage
		liveRequest := LiveRequest{
			RequestBody: &body,
		}

		if err := conn.ReadJSON(&liveRequest); err != nil {
			log.Println(err)
		}

		switch liveRequest.RequestName {
		case "install":
			var installRequest InstallRequest
			if err := json.Unmarshal(body, &installRequest); err != nil {
				log.Println(err)
			}
			var distro string = installRequest.Distro
			var packages string = installRequest.Packages

			installCommand := ""
			if strings.Contains(distro, "fedora") || strings.Contains(distro, "centos") {
				installCommand = "yum -y install " + packages
			} else {
				installCommand = "apt-get -y install " + packages
			}
			session.runLiveCommand(installCommand)
		case "remove":
			var installRequest InstallRequest
			if err := json.Unmarshal(body, &installRequest); err != nil {
				log.Println(err)
			}
			var distro string = installRequest.Distro
			var packages string = installRequest.Packages

			installCommand := ""
			if strings.Contains(distro, "fedora") || strings.Contains(distro, "centos") {
				installCommand = "yum -y --skip-broken remove " + packages
			} else {
				installCommand = "apt-get -y remove " + packages
			}
			session.runLiveCommand(installCommand)
		case "logfile":
			var logFileRequest LogFileRequest
			if err := json.Unmarshal(body, &logFileRequest); err != nil {
				log.Println(err)
			}
			
			var command string = logFileRequest.Command
			var path string = logFileRequest.Path
			var options string = logFileRequest.Options
			
			logfileCommand := command + " " + options + " " + path
			log.Println(logfileCommand)
			session.runLiveCommand(logfileCommand)
		}
	}

	log.Println("Exiting handleWebSocket")
}

func (ses *webSocketSession) runLiveCommand(command string) {

	cmd := exec.Command("bash", "-c", command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Println(err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Println(err)
		return
	}

	s := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for s.Scan() {
		log.Println(string(s.Bytes()))
		ses.connection.WriteMessage(1, s.Bytes())
	}

	if err := cmd.Wait(); err != nil {
		log.Println(err)
		return
	}

	ses.connection.WriteMessage(1, []byte("Finished\n"))
}
