package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type GroupUpdate struct {
	Username string
	Groups   []string
}

func handleGetFile(w http.ResponseWriter, r *http.Request) {
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)

	file, err := ioutil.ReadFile(p["path"])
	if err != nil {
		w.WriteHeader(404)
	}
	stringFile := string(file)
	w.Write([]byte(stringFile))
}

func handleGetUsersGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cmd := exec.Command("/usr/bin/id", "-Gn", vars["username"])
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	w.Write([]byte(out.String()))
}

func handleUpdateGroups(w http.ResponseWriter, r *http.Request) {
	var p GroupUpdate
	err := json.NewDecoder(r.Body).Decode(&p)

	groupsFile, err := ioutil.ReadFile("/etc/group")
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(500)
	}

	username := p.Username
	groupMembers := p.Groups

	groupsFinal := processGroupFile(username, string(groupsFile), groupMembers)

	writeErr := ioutil.WriteFile("/etc/group", []byte(groupsFinal), 544)
	if writeErr != nil {
		log.Fatal(writeErr)
		w.WriteHeader(500)
	}
	w.WriteHeader(200)
}

func processGroupFile(username string, groupFile string, groupMembers []string) string {
	// this is the final result we'll be building up and passing back
	resultString := ""

	// break the groups into lines
	lines := strings.FieldsFunc(groupFile, func(r rune) bool {
		if r == '\n' {
			return true
		}
		return false
	})

	// break the lines into their four tokens (groupname, x, gid, users array)
	for _, line := range lines {
		thisLineString := ""
		ss := strings.FieldsFunc(line, func(r rune) bool {
			if r == ':' {
				return true
			}
			return false
		})

		groupName := strings.TrimSpace(ss[0])
		groupX := strings.TrimSpace(ss[1])
		groupId := strings.TrimSpace(ss[2])
		groupUsers := ""
		if len(ss) == 4 {
			groupUsers = ss[3]
		}

		// break the users list into an array
		groupUserArray := strings.FieldsFunc(groupUsers, func(r rune) bool {
			if r == ',' {
				return true
			}
			return false
		})

		// Process a line if it's a group that we're changing
		_, lineMatters := FindInArray(groupMembers, groupName)
		if lineMatters {
			_, alreadyIncludes := FindInArray(groupUserArray, username)
			if !alreadyIncludes {
				if len(groupUsers) > 0 {
					thisLineString = groupName + ":" + groupX + ":" + groupId + ":" + groupUsers + "," + username
				} else {
					thisLineString = groupName + ":" + groupX + ":" + groupId + ":" + username
				}

			} else {
				// just leave the line unchanged
				thisLineString = groupName + ":" + groupX + ":" + groupId + ":" + groupUsers
			}
		} else {
			// if a user should be removed from a group, do that now
			foundIndex, existingGroupMember := FindInArray(groupUserArray, username)
			if existingGroupMember {
				groupUserArray = append(groupUserArray[:foundIndex], groupUserArray[foundIndex+1:]...)
				thisLineString = groupName + ":" + groupX + ":" + groupId + ":" + strings.Join(groupUserArray, ",")
			} else {
				thisLineString = groupName + ":" + groupX + ":" + groupId + ":" + groupUsers
			}
		}
		resultString += thisLineString + "\n"
	}
	return resultString
}

func FindInArray(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}
