package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type FileRequest struct {
	RequestType string
	RequestBody interface{}
}

// RequestType "directoryList"
type DirectoryRequest struct {
	DirectoryPath string
}

type FileInfo struct {
	Path    string      // path to file (provided by request)
	Name    string      // base name of the file
	Size    int64       // length in bytes for regular files; system-dependent for others
	Mode    os.FileMode // file mode bits
	ModTime time.Time   // modification time
	Sys     interface{} // underlying info? I don't know.
	IsDir   bool
}

func (app *webSocketApp) handleFileOperations(w http.ResponseWriter, r *http.Request) {
	log.Println("This is /fileOperations starting")

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
		fileRequest := FileRequest{
			RequestBody: &body,
		}

		if err := conn.ReadJSON(&fileRequest); err != nil {
			log.Fatal(err)
		}

		switch fileRequest.RequestType {
		case "directoryList":
			var directoryRequest DirectoryRequest
			if err := json.Unmarshal(body, &directoryRequest); err != nil {
				log.Fatal(err)
			}
			var path string = directoryRequest.DirectoryPath
			log.Println("Display directory for " + path)
			directoryContents, err := ioutil.ReadDir(path)
			if err != nil {
				log.Println(err)
			}

			var contentsInfo []FileInfo
			for _, info := range directoryContents {

				var isThisDir = info.IsDir()

				// if info.Mode()&os.ModeSymlink != 0 {
				// 					originFile, err := os.Readlink(info.Name())
				// 					if err != nil {
				// 						log.Println(err)
				// 					}
				//
				// 					isThisDir = originFile.IsDir()
				// 				}

				contentsInfo = append(contentsInfo, FileInfo{
					Path:    path,
					Name:    info.Name(),
					Size:    info.Size(),
					Mode:    info.Mode().Perm(),
					ModTime: info.ModTime(),
					IsDir:   isThisDir,
					Sys:     info.Sys(),
				})
			}

			js, err := json.Marshal(contentsInfo)
			if err != nil {
				log.Println(err)
			}

			session.connection.WriteMessage(1, js)

		}
	}

	log.Println("Exiting handleWebSocket")
}
