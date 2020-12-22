package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"strconv"
)

type GroupUpdate struct {
	Username string
	Groups   []string
}

type NewUser struct {
	Username   string
	Fullname   string
	Password   string
	Shell      string
	HasHomeDir bool
}

func handleGetFile(w http.ResponseWriter, r *http.Request) {
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)
	
	u, err := user.Current()
	if err != nil {
		log.Printf("Error getting user: %s", err)
		return
	}
	log.Println("User is ", u.Uid)

	file, err := ioutil.ReadFile(p["path"])
	if err != nil {
		u, err := user.Current()
		if err != nil {
			log.Printf("Error getting user: %s", err)
			return
		}
		log.Println("While fetching %s as user %s", p["path"], u.Uid)
		log.Println("Error: %s", err)
		w.WriteHeader(404)
	}
	stringFile := string(file)
	w.Write([]byte(stringFile))
}

func handleChownFile(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling chown file...")
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)
	
	log.Println("Receiving %+v\n", p)
	
	path := p["path"]
	uid, err := strconv.Atoi(p["uid"]); if err != nil { w.WriteHeader(500); return }
	gid, err := strconv.Atoi(p["gid"]); if err != nil { w.WriteHeader(500); return }
	
	log.Println("Getting %d, %d", uid, gid)
	
	err = os.Chown(path, uid, gid)
	if err != nil {
		w.WriteHeader(500)
	}
	w.WriteHeader(200)
}

func handleChmodFile(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling chmod file...")
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	
	path := p["path"]
	
	log.Println("Received mode: " + p["mode"])
	modeVal, _ := strconv.ParseUint(p["mode"], 8, 32)
	log.Println("Mode:", modeVal)
	newMode := os.FileMode(modeVal)
	log.Println("FileMode: ", newMode)
	if err := os.Chmod(path, newMode); err != nil {
		w.WriteHeader(500)
	}
	w.WriteHeader(200)
}

func handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling downloadFile")
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.Println(err)
	}
	path := p["path"]
	
	log.Println("Got path: " + path)
	http.ServeFile(w, r, path)
}

func handleUploadFile(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling uploadFile")
	
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("path")
	if err != nil {
		log.Println("Error Retrieving the File")
		log.Println(err)
		return
	}
	defer file.Close()
	log.Printf("Uploaded File: %+v\n", handler.Filename)
	log.Printf("File Size: %+v\n", handler.Size)
	log.Printf("MIME Header: %+v\n", handler.Header)

	// Create file
	dst, err := os.Create(handler.Filename)
	defer dst.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(200)
}

func handleGetCron(w http.ResponseWriter, r *http.Request) {
	argusCronPath := "/etc/cron.d/argus"
	cronExists, _ := exists("/etc/cron.d")
	cronFileExists, _ := exists(argusCronPath)
	if !cronExists {
		w.WriteHeader(500) // cron not installed?
	} else {
		if !cronFileExists { // first time, create our file
			emptyFile, err := os.Create(argusCronPath)
			if err != nil {
				log.Fatal(err)
			}
			emptyFile.Close()
		}
	}
	
	log.Println("Get contents for " + argusCronPath)
	crontab, err := ioutil.ReadFile(argusCronPath)
	if err != nil {
		log.Println(err)
	}
	w.Write(crontab)	
}

func handleSetCron(w http.ResponseWriter, r *http.Request) {
	argusCronPath := "/etc/cron.d/argus"
	
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.Println(err)
	}
	
	stringBytes := []byte(p["crontab"])
	
	err2 := ioutil.WriteFile(argusCronPath, stringBytes, 0644)
	if err2 != nil {
		log.Println(err2)
	}
	
	w.WriteHeader(200)
}

func handleGetUsersGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cmd := exec.Command("/usr/bin/id", "-Gn", vars["username"])
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println(err)
	}
	w.Write([]byte(out.String()))
}

func handleUpdateGroups(w http.ResponseWriter, r *http.Request) {
	var p GroupUpdate
	err := json.NewDecoder(r.Body).Decode(&p)

	groupsFile, err := ioutil.ReadFile("/etc/group")
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
	}

	username := p.Username
	groupMembers := p.Groups

	groupsFinal := processGroupFile(username, string(groupsFile), groupMembers)

	writeErr := ioutil.WriteFile("/etc/group", []byte(groupsFinal), 544)
	if writeErr != nil {
		log.Println(writeErr)
		w.WriteHeader(500)
	}
	w.WriteHeader(200)
}

func handleNewUser(w http.ResponseWriter, r *http.Request) {
	var n NewUser
	err := json.NewDecoder(r.Body).Decode(&n)
	if err != nil {
		log.Println(err)
	}

	commands := []string{"-c", n.Fullname}
	if n.HasHomeDir {
		commands = append(commands, "-m")
	}
	commands = append(commands, "-s", n.Shell)
	saltedPass := hashAndSalt([]byte(n.Password))
	commands = append(commands, "--password", saltedPass)
	commands = append(commands, n.Username)
	log.Println(commands)
	cmd := exec.Command("/usr/sbin/useradd", commands...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err2 := cmd.Run()
	if err2 != nil {
		log.Println(err2)
	}
	w.Write([]byte(out.String()))
}

func handleRemoveUser(w http.ResponseWriter, r *http.Request) {
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.Println(err)
	}
	user := p["username"]

	// kill all current tasks by user
	killCommand := exec.Command("/usr/bin/killall", "-KILL", "-u", user)
	killCommand.Run()

	// remove cron jobs
	cronRemove := exec.Command("crontab", "-r", "-u", user)
	cronRemove.Run()

	// backup the home folder
	backupCommand := exec.Command("tar", "-zcvf", "/root/"+user+".backup.tar.gz", "/home/"+user)
	backupCommand.Run()

	// delete the account
	delCommand := exec.Command("userdel", "-r", user)
	var out bytes.Buffer
	delCommand.Stdout = &out

	err2 := delCommand.Run()
	if err2 != nil {
		log.Println(err2)
	}
	w.Write([]byte(out.String()))

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

func hashAndSalt(pwd []byte) string {
	// FROM https://medium.com/@jcox250/password-hash-salt-using-golang-b041dc94cb72

	// Use GenerateFromPassword to hash & salt pwd
	// MinCost is just an integer constant provided by the bcrypt
	// package along with DefaultCost & MaxCost.
	// The cost can be any value you want provided it isn't lower
	// than the MinCost (4)
	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.MinCost)
	if err != nil {
		log.Println(err)
	}
	// GenerateFromPassword returns a byte slice so we need to
	// convert the bytes to a string and return it
	return string(hash)
}

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return false, err
}
