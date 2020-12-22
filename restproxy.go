package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

// ProxyRequests allow argusd to work with other servers operating on the same machine
// initially intended to work with the CaddyServer API
// the caller provides a JSON struct with the request to pass along; this will make 
// the request and return the result as transparently as I can figure out.

type ProxyRequest struct {
	Port int
	Method string
	Path string
	Body string
	Headers map[string]string
}

func handleRestProxy(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting REST Proxy Request...")
	var req ProxyRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	
	if err != nil {
		log.Println("Error with Rest Proxy request: ", err)
		w.WriteHeader(500)
	}
		
	client := &http.Client{}
	
	path := "http://localhost" + ":" + strconv.Itoa(req.Port) + req.Path 
	log.Println("Sending request to ", path)
	
	var httpRequest *http.Request
	if len(req.Body) > 0 {
		jsonBody := []byte(req.Body)
		httpRequest, err = http.NewRequest(req.Method, path, bytes.NewBuffer(jsonBody))
	} else {
		httpRequest, err = http.NewRequest(req.Method, path, nil)
	}
	
	for k, v := range req.Headers {
		httpRequest.Header.Add(k, v)
	}
	
	resp, err := client.Do(httpRequest)
	if err != nil {
		log.Println("Error: ", err)
		w.WriteHeader(500)
	}
	// defer resp.Body.Close()
	
	status := resp.StatusCode
	body, _ := ioutil.ReadAll(resp.Body)
	
	w.WriteHeader(status)
	w.Write([]byte(body))
}