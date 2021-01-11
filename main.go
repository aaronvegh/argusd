package main

import (
	"encoding/json"
	"io/ioutil"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/creack/golisten"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/lestrrat-go/file-rotatelogs"
	
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

	// websocket endpoints
	app.router.HandleFunc("/systemStatus", app.handleSystemStatus).Methods("GET")
	app.router.HandleFunc("/dashboard", app.handleDashboard).Methods("GET")
	app.router.HandleFunc("/liveResponse", app.handleLiveResponse).Methods("GET")
	app.router.HandleFunc("/fileOperations", app.handleFileOperations).Methods("GET")
	
	// REST endpoints
	app.router.HandleFunc("/whoami", handleWhoAmI).Methods("GET")
	app.router.HandleFunc("/getVersion", handleGetVersion).Methods("GET")
	app.router.HandleFunc("/getFile", app.handleGetFile).Methods("POST")
	app.router.HandleFunc("/chown", app.handleChownFile).Methods("POST")
	app.router.HandleFunc("/chmod", app.handleChmodFile).Methods("POST")
	app.router.HandleFunc("/downloadFile", app.handleDownloadFile).Methods("POST")
	app.router.HandleFunc("/uploadFile", app.handleUploadFile).Methods("POST")
	app.router.HandleFunc("/getCron", app.handleGetCron).Methods("GET")
	app.router.HandleFunc("/setCron", app.handleSetCron).Methods("POST")
	app.router.HandleFunc("/getUsersGroups/{username}", app.handleGetUsersGroups).Methods("GET")
	app.router.HandleFunc("/updateGroups", app.handleUpdateGroups).Methods("POST")
	app.router.HandleFunc("/newUser", app.handleNewUser).Methods("POST")
	app.router.HandleFunc("/removeUser", app.handleRemoveUser).Methods("POST")
	app.router.HandleFunc("/packages/all/{distro}", app.handleInstalledPackages).Methods("GET")
	app.router.HandleFunc("/packages/getInfo", app.handlePackageInfo).Methods("POST")
	app.router.HandleFunc("/packages/search", app.handleFindPackages).Methods("POST")
	app.router.HandleFunc("/packages/upgradable/{distro}", app.handleUpgradable).Methods("GET")
	app.router.HandleFunc("/restProxy", app.handleRestProxy).Methods("POST")
	app.router.HandleFunc("/getCaddyConfig", app.handleGetCaddyConfig).Methods("GET")
	app.router.HandleFunc("/setCaddyConfig", app.handleSetCaddyConfig).Methods("POST")
	
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

func handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting whoami handler")
	u, err := user.Current()
	if err != nil {
		log.Printf("Error getting user: %s", err)
		return
	}
	fmt.Fprintf(w, "%s\n", u.Uid)
}

func AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuidMatcher := `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`
		if token := r.Header.Get("X-Argus-Token"); token != "" {
			matched, err := regexp.MatchString(uuidMatcher, token)
			if err != nil {
				// bad regex, so this shouldn't happen right? RIGHT?
				log.Fatal("Error, fix your auth regex!")
			}
			if !matched {
				log.Println("Error: submitted token is not a valid UUID")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			
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
			log.Println("Error: Token doesn't match.")
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			log.Println("Error: Token not present in headers.")
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	})
}

var (
	Version = ""
)

func main() {
	// Make the logs go brrr
	logf, err := rotatelogs.New(
		"/root/.arguslogs/log.%Y%m%d%H%M",
		rotatelogs.WithLinkName("/root/.arguslogs/log"),
		rotatelogs.WithMaxAge(24 * time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)
  	if err != nil {
	  log.Printf("failed to create rotatelogs: %s", err)
	  return
  	}
	log.SetOutput(logf)
	
	// establish the NonRootUser for the unprivileged process
	os.Setenv("TMPDIR", "/var/tmp/")
	nonRootUser := os.Getenv("NonRootUser")	
	log.Println("User from env: ", nonRootUser)
	if len(nonRootUser) == 0 {
		nonRootUser = ""
	}
	
	// get the current user to ensure this runs as root
	u, err := user.Current()
	if err != nil {
		log.Printf("Error getting user: %s", err)
		return
	}
	
	// Create a WaitGroup to manage the four servers
	// https://medium.com/rungo/running-multiple-http-servers-in-go-d15300f4e59f
	wg := new(sync.WaitGroup)
	if u.Uid == "0" && nonRootUser != "" {
		wg.Add(4)
	} else {
		wg.Add(2)
	}
	
	app, err := newWebSocketApp()
	if err != nil {
		log.Fatal("Could not create WebSocketApp:", err)
	}

	
	if u.Uid == "0" { // I'm root!
		go func() {
			for {
				autoUpdate()
				time.Sleep(10 * time.Minute)
			}
		}()
	
		// Setup our Ctrl+C handler
		SetupCloseHandler()
		
		// standard root websocket application
		go func() {
			log.Println("Mounting root REST server at port 26510.")
			server := &http.Server{
				Handler:      app.router,
				WriteTimeout: 15 * time.Second,
				ReadTimeout:  15 * time.Second,
				Addr:         ":26510",
			}
			if err := server.ListenAndServe(); err != nil {
				log.Fatal("Could not listen and serve privileged app: ", err)
			}
			wg.Done()
		}()
	}
	
	if nonRootUser != "" {
		go func() {
			log.Println("Mounting non-privilege REST server at port 26511.")
			if err := golisten.ListenAndServe("aaron", "127.0.0.1:26511", app.router); err != nil {
				log.Println("Could not listen and serve non-privileged app: ", err)
			}
			wg.Done()
		}()
	}
	
	wg.Wait()

}
