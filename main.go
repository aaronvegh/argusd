package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"os"
	"os/signal"
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

	app := &webSocketApp{
		upgrader: upgrader,
		router:   mux.NewRouter(),
	}

	app.router.HandleFunc("/systemStatus", app.handleSystemStatus).Methods("GET")
	app.router.HandleFunc("/dashboard", app.handleDashboard).Methods("GET")

	return app, nil
}

func (app *webSocketApp) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	log.Println("This is /systemStatus starting")

	connection, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	session := &webSocketSession{
		connection: connection,
	}

	session.runSystemStatus()
}

func (app *webSocketApp) handleDashboard(w http.ResponseWriter, r *http.Request) {
	log.Println("This is /connect starting")

	connection, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	session := &webSocketSession{
		connection: connection,
	}

	session.runDashboard()
}

func (app *webSocketApp) run() error {
	srv := &http.Server{
		Handler:      app.router,
		Addr:         ":26510",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return srv.ListenAndServe()
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

func main() {

	// Setup our Ctrl+C handler
	SetupCloseHandler()

	restApp := mux.NewRouter()
	restApp.HandleFunc("/getFile", handleGetFile).Methods("POST")
	restApp.HandleFunc("/getUsersGroups/{username}", handleGetUsersGroups).Methods("GET")
	restApp.HandleFunc("/updateGroups", handleUpdateGroups).Methods("POST")
	go http.ListenAndServe(":26511", restApp)

	app, err := newWebSocketApp()
	if err != nil {
		log.Fatal("Could not create app:", err)
	}

	if err := app.run(); err != nil {
		log.Fatal("Could not run app:", err)
	}

}
