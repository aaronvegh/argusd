package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"
	"github.com/machinebox/progress"
)

type FileRequest struct {
	RequestType string
	RequestBody interface{}
}

type FileResponse struct {
	ResponseType string
	ResponseBody interface{}
}

// RequestType "directoryList"
type DirectoryRequest struct {
	DirectoryPath string
}

// RequestType "fileContents"
type FileContentRequest struct {
	FilePath string
}

// RequestType "moveFile"
type MoveFileRequest struct {
	OriginPath string
	DestinationPath string
}

// RequestType "copyFile"
type CopyFileRequest struct {
	OriginPath string
	DestinationPath string
}

// RequestType "deleteFile"
type DeleteFileRequest struct {
	FilePath string
}

type FileOperationProgress struct {
	PercentRemaining time.Duration
}

type FileContentsBody struct {
	RequestPath     string
	RequestContents []byte
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
			log.Println(err)
		}

		switch fileRequest.RequestType {
		case "fileContents":
			log.Println("Getting filecontents...")
			var fileRequest FileContentRequest
			if err := json.Unmarshal(body, &fileRequest); err != nil {
				log.Println(err)
			}
			var path string = fileRequest.FilePath
			log.Println("Get contents for " + path)
			fileContents, err := ioutil.ReadFile(path)
			if err != nil {
				log.Println(err)
			}

			fileContentsResp := FileContentsBody{
				RequestPath:     path,
				RequestContents: fileContents,
			}

			response := FileResponse{
				ResponseType: "fileContents",
				ResponseBody: fileContentsResp,
			}

			js, err := json.Marshal(response)
			if err != nil {
				log.Println(err)
			}

			session.connection.WriteMessage(1, js)

		case "directoryList":
			log.Println("Getting directoryList...")
			var directoryRequest DirectoryRequest
			if err := json.Unmarshal(body, &directoryRequest); err != nil {
				log.Println(err)
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

			response := FileResponse{
				ResponseType: "directoryList",
				ResponseBody: contentsInfo,
			}

			js, err := json.Marshal(response)
			if err != nil {
				log.Println(err)
			}

			session.connection.WriteMessage(1, js)
			
		case "copyFile":
			log.Println("Getting copyfile...")
			var copyFileRequest CopyFileRequest
			if err := json.Unmarshal(body, &copyFileRequest); err != nil {
				log.Println(err)
				continue
			}
			var origin string = copyFileRequest.OriginPath
			var destination string = copyFileRequest.DestinationPath
			
			log.Println("origin: " + origin)
			log.Println("dest: " + destination)
			
			copyFile(session, origin, destination)
			  
		case "moveFile":
			log.Println("Getting moveFile...")
			var moveFileRequest MoveFileRequest
			if err := json.Unmarshal(body, &moveFileRequest); err != nil {
				log.Println(err)
				return
			}
			var origin string = moveFileRequest.OriginPath
			var destination string = moveFileRequest.DestinationPath
			
			log.Println("origin: " + origin)
			log.Println("dest: " + destination)
			
			from, err := os.Open(origin)
			if err != nil {
				log.Println(err)
				return
			}
			defer from.Close()
			
			ctx := context.Background()
			r := progress.NewReader(from)
			fileInfo, err := from.Stat()
			if err != nil {
				log.Println(err)
				return
			}
			size := fileInfo.Size()
			
			go func() {
				progressChan := progress.NewTicker(ctx, r, size, 1*time.Second)
				for p := range progressChan {
					fileOpProgress := FileOperationProgress {
						PercentRemaining: p.Remaining().Round(time.Second),
					}
					
					response := FileResponse {
						ResponseType: "fileProgress",
						ResponseBody: fileOpProgress,
					}
					
					js, err := json.Marshal(response)
					if err != nil {
						log.Println(err)
						return
					}
					log.Println(string(js))
					session.connection.WriteMessage(1, js)
				}
				log.Println("\rMove is completed")
			}()
			
			err = os.Rename(origin, destination)
			if err != nil {
				if err, ok := err.(*os.LinkError); ok {
					oserr := err.Err.(syscall.Errno)
					if oserr == syscall.EXDEV {
						log.Println("Failed because you are copying a file cross filesystems; copy instead\n")
						copyFile(session, origin, destination)
					} else {
						log.Println("Unknown OS Error is %d\n", oserr)
					}
					return
				}
			}
		case "deleteFile":
			log.Println("Getting deleteFile...")
			var deleteFileRequest DeleteFileRequest
			if err := json.Unmarshal(body, &deleteFileRequest); err != nil {
				log.Println(err)
			}
			var filePath string = deleteFileRequest.FilePath
			
			log.Println("filePath: " + filePath)
		
			err = os.Remove(filePath)
			if err != nil {
				log.Println(err)
			}
			
			response := FileResponse {
				ResponseType: "deleteFile",
				ResponseBody: "ok",
			}
			
			js, err := json.Marshal(response)
			if err != nil {
				log.Println(err)
			}
			
			session.connection.WriteMessage(1, js)
		}		
	}

	log.Println("Exiting handleWebSocket")
}

func copyFile(session *webSocketSession, origin string, destination string) {
	from, err := os.Open(origin)
	if err != nil {
		log.Println(err)
		return
	}
	defer from.Close()
	
	ctx := context.Background()
	r := progress.NewReader(from)
	fileInfo, err := from.Stat()
	if err != nil {
		log.Println(err)
		return
	}
	size := fileInfo.Size()
	
	go func() {
		progressChan := progress.NewTicker(ctx, r, size, 1*time.Second)
		for p := range progressChan {
			fileOpProgress := FileOperationProgress {
				PercentRemaining: p.Remaining().Round(time.Second),
			}
			
			response := FileResponse {
				ResponseType: "fileProgress",
				ResponseBody: fileOpProgress,
			}
			
			js, err := json.Marshal(response)
			if err != nil {
				log.Println(err)
			}
			log.Println(string(js))
			session.connection.WriteMessage(1, js)
		}
		log.Println("\rdownload is completed")
	}()
	
	to, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
	}
	defer to.Close()
	
	_, err = io.Copy(to, from)
	if err != nil {
		log.Println(err)
	}
}
