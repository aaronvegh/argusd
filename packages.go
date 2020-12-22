package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"
)

type InstalledPackage struct {
	PackageName    string
	PackageVersion string
}

type SearchResultPackage struct {
	PackageName        string
	PackageDescription string
}

func handleInstalledPackages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	distro := vars["distro"]

	listCommand := ""
	if strings.Contains(distro, "fedora") || strings.Contains(distro, "centos") {
		listCommand = "yum list installed | awk 'NR>1 {print $1,$2}'"
	} else {
		listCommand = "dpkg -l | awk 'NR>5{print $2,$3}'"
	}

	cmd := exec.Command("bash", "-c", listCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Println(cmdErr)
	}

	lines := strings.FieldsFunc(out.String(), func(r rune) bool {
		if r == '\n' {
			return true
		}
		return false
	})

	var installedPackages []InstalledPackage
	for i, line := range lines {
		if i == 0 {
			continue
		}
		item := strings.FieldsFunc(line, func(r rune) bool {
			if r == ' ' {
				return true
			}
			return false
		})
		packageName := item[0]
		packageVersion := ""
		if len(item) > 1 {
			packageVersion = item[1]
		}
		installedPackages = append(installedPackages, InstalledPackage{
			PackageName:    packageName,
			PackageVersion: packageVersion,
		})
	}

	js, err := json.Marshal(installedPackages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}

func handlePackageInfo(w http.ResponseWriter, r *http.Request) {
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.Println(err)
	}

	distro := p["distro"]
	packageName := p["package"]

	listCommand := ""
	if strings.Contains(distro, "fedora") || strings.Contains(distro, "centos") {
		listCommand = "yum info " + packageName
	} else {
		listCommand = "apt-cache show " + packageName
	}

	cmd := exec.Command("bash", "-c", listCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Println(cmdErr)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(out.String()))
}

func handleFindPackages(w http.ResponseWriter, r *http.Request) {
	var p map[string]string
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.Println(err)
	}

	distro := p["distro"]
	query := p["query"]

	listCommand := ""
	if strings.Contains(distro, "fedora") || strings.Contains(distro, "centos") {
		listCommand = "yum search " + query + "| awk -F' : ' 'BEGIN{OFS=\"|\";} /^([^:]*?) : (.*)$/ {print $1, $2}'"
	} else {
		listCommand = "apt-cache search " + query + "| awk -F ' - ' 'BEGIN{OFS=\"|\";} {print $1, $2}'"
	}

	cmd := exec.Command("bash", "-c", listCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Println(cmdErr)
	}

	lines := strings.FieldsFunc(out.String(), func(r rune) bool {
		if r == '\n' {
			return true
		}
		return false
	})

	var resultPackages []SearchResultPackage
	for _, line := range lines {
		fields := strings.FieldsFunc(line, func(r rune) bool {
			if r == '|' {
				return true
			}
			return false
		})

		packageName := fields[0]
		packageDesc := ""
		if len(fields) > 1 {
			packageDesc = fields[1]
		}
		resultPackages = append(resultPackages, SearchResultPackage{
			PackageName:        packageName,
			PackageDescription: packageDesc,
		})

	}

	js, err := json.Marshal(resultPackages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}

func handleUpgradable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	distro := vars["distro"]

	listCommand := ""
	if strings.Contains(distro, "fedora") || strings.Contains(distro, "centos") {
		listCommand = "yum -q check-update | awk '{print $1,$2}'"
	} else {
		listCommand = "apt-get --simulate upgrade | awk '/^Inst ([a-zA-Z0-9\\-\\.]+) \\[([a-zA-Z0-9\\-\\.\\:]+)\\]/{print $2,$3}'"
		exec.Command("bash", "-c", "apt update")
	}

	cmd := exec.Command("bash", "-c", listCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Println(cmdErr)
	}

	lines := strings.FieldsFunc(out.String(), func(r rune) bool {
		if r == '\n' {
			return true
		}
		return false
	})

	var upgradablePackages []InstalledPackage
	for i, line := range lines {
		if i == 0 {
			continue
		}
		item := strings.FieldsFunc(line, func(r rune) bool {
			if r == ' ' {
				return true
			}
			return false
		})
		packageName := item[0]
		packageVersion := ""
		if len(item) > 1 {
			packageVersion = item[1]
			packageVersion = strings.Replace(packageVersion, "[", "", -1)
			packageVersion = strings.Replace(packageVersion, "]", "", -1)
		}
		upgradablePackages = append(upgradablePackages, InstalledPackage{
			PackageName:    packageName,
			PackageVersion: packageVersion,
		})
	}

	js, err := json.Marshal(upgradablePackages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}


func (ses *webSocketSession) runPackageInstall(distro string, packages string) {

	installCommand := ""
	if strings.Contains(distro, "fedora") || strings.Contains(distro, "centos") {
		installCommand = "yum -y install " + packages
	} else {
		installCommand = "apt-get -y install " + packages
	}

	log.Println("Install Command" + installCommand)
	cmd := exec.Command("bash", "-c", installCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Println(cmdErr)
		return
	}

	err := ses.connection.WriteMessage(1, []byte(out.String()))
	if err != nil {
		log.Println("write:", err)
		return
	}
}
