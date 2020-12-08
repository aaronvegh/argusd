package main

import (
	"encoding/json"
	"io/ioutil"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/creack/golisten"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"os"
	"os/signal"
	"os/user"
	"strings"
	"sync"
	"syscall"
)

type webSocketSession struct {
	connection *websocket.Conn
}

type dict map[string]interface{}

type payload struct {
	name string
	ptr  dict
}

type webSocketApp struct {
	router   *mux.Router
	upgrader websocket.Upgrader
}

func newWebSocketApp() (*webSocketApp, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	router := mux.NewRouter()
	router.Use(AuthenticationMiddleware)

	app := &webSocketApp{
		upgrader: upgrader,
		router:   router,
	}

	app.router.HandleFunc("/systemStatus", app.handleSystemStatus).Methods("GET")
	app.router.HandleFunc("/dashboard", app.handleDashboard).Methods("GET")
	app.router.HandleFunc("/liveResponse", app.handleLiveResponse).Methods("GET")
	app.router.HandleFunc("/fileOperations", app.handleFileOperations).Methods("GET")

	return app, nil
}

// SetupCloseHandler creates a 'listener' on a new goroutine which will notify the
// program if it receives an interrupt from the OS. We then handle this by calling
// our clean up procedure and exiting the program.
func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("\r- Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}

func autoUpdate() {
	err := AutoUpdate(Updater{
		CurrentVersion: Version,
		S3Bucket:       "argusd",
		S3Region:       "us-east-1",
		S3ReleaseKey:   "argusd/argusd-{{OS}}-{{ARCH}}",
		S3VersionKey:   "argusd/VERSION",
	})
	if err != nil {
		log.Println(err)
	}
}

func handleGetVersion(w http.ResponseWriter, r *http.Request) {
	versionString := dict{
		"version": Version,
	}

	js, err := json.Marshal(versionString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := r.Header.Get("X-Argus-Token"); token != "" {
			file, err := ioutil.ReadFile("/etc/argusd.conf")
			if err != nil {
				log.Println("Error: Can't read config file")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			serverTokens := string(file)
			tokenLines := strings.Split(serverTokens, "\n")
			index, _ := FindInArray(tokenLines, token)
			if index >= 0 {
				next.ServeHTTP(w, r)
				return
			}
		}
		log.Println("Error: Token doesn't match.")
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

var (
	Version = ""
)

func restHandler() http.Handler {
	restApp := mux.NewRouter()
	restApp.Use(AuthenticationMiddleware)
	restApp.HandleFunc("/getVersion", handleGetVersion).Methods("GET")
	restApp.HandleFunc("/getFile", handleGetFile).Methods("POST")
	restApp.HandleFunc("/chown", handleChownFile).Methods("POST")
	restApp.HandleFunc("/chmod", handleChmodFile).Methods("POST")
	restApp.HandleFunc("/downloadFile", handleDownloadFile).Methods("POST")
	restApp.HandleFunc("/uploadFile", handleUploadFile).Methods("POST")
	restApp.HandleFunc("/getUsersGroups/{username}", handleGetUsersGroups).Methods("GET")
	restApp.HandleFunc("/updateGroups", handleUpdateGroups).Methods("POST")
	restApp.HandleFunc("/newUser", handleNewUser).Methods("POST")
	restApp.HandleFunc("/removeUser", handleRemoveUser).Methods("POST")
	restApp.HandleFunc("/packages/all/{distro}", handleInstalledPackages).Methods("GET")
	restApp.HandleFunc("/packages/getInfo", handlePackageInfo).Methods("POST")
	restApp.HandleFunc("/packages/search", handleFindPackages).Methods("POST")
	restApp.HandleFunc("/packages/upgradable/{distro}", handleUpgradable).Methods("GET")
	return restApp
}

func createRestServer(port int) *http.Server {
	server := http.Server {
		Addr: fmt.Sprintf( "127.0.0.1:%v", port),
		Handler: restHandler(),
	}
	return &server
}

func main() {	
	os.Setenv("TMPDIR", "/var/tmp/")
	nonRootUser := os.Getenv("NonRootUser")	
	
	go func() {
		for {
			autoUpdate()
			time.Sleep(10 * time.Minute)
		}
	}()

	// Setup our Ctrl+C handler
	SetupCloseHandler()

	app, err := newWebSocketApp()
	if err != nil {
		log.Println("Could not create app:", err)
	}
	
	u, err := user.Current()
	if err != nil {
		log.Printf("Error getting user: %s", err)
		return
	}
	
	// Create a WaitGroup to manage the four servers
	// https://medium.com/rungo/running-multiple-http-servers-in-go-d15300f4e59f
	wg := new(sync.WaitGroup)
	wg.Add(2)
	
	if u.Uid == 0 { // I'm root!
		// standard root websocket application
		go func() {
			rootSrv := &http.Server{
				Handler:      app.router,
				Addr:         "127.0.0.1:26510",
				WriteTimeout: 15 * time.Second,
				ReadTimeout:  15 * time.Second,
			}
			rootSrv.ListenAndServe()
			wg.Done()
		}()
		
		// standard root REST application
		go func() {
			server := createRestServer(26511)
			if err := server.ListenAndServe(); err != nil {
				log.Println("Could not listen and serve: ", err)
			}
			wg.Done()
		}()
	} else {
		// Non-privileged websocket application	
		go func() {
			nonRootSrv := &http.Server{
				Handler:      app.router,
				WriteTimeout: 15 * time.Second,
				ReadTimeout:  15 * time.Second,
			}
			ln, _ := golisten.Listen(nonRootUser, "tcp", "127.0.0.1:26610")
			nonRootSrv.Serve(ln)
			wg.Done()	
		}()
		
		// Non-privileged REST application
		go func() {
			if err := golisten.ListenAndServe(nonRootUser, "127.0.0.1:26611", restHandler()); err != nil {
				log.Println("Could not listen and serve: ", err)
			}
			wg.Done()
		}()
	}
	
	wg.Wait()

}
