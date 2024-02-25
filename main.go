package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	listenAddr       = ":8086"
	influxDBServer   = "https://eu-central-1-1.aws.cloud.influxdata.com:443"
)

func main() {
	// Start the proxy server
	http.HandleFunc("/", handleRequest)
	log.Printf("Starting InfluxDB proxy server on %s...\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Log the incoming request
	log.Printf("Received request: %s %s %s\n", r.Method, r.URL.Path, r.URL.RawQuery)

	log.Printf("Request Header: %s\n", r.Header)

	// Decode and log the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	log.Printf("Request Body: %s\n", body)

	// Restore the request body so it can be forwarded
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Measure time to execute the query
	startTime := time.Now()

	// Forward the request to InfluxDB server
	resp, err := forwardRequest(r)
	if err != nil {
		log.Printf("Error forwarding request: %v\n", err)
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Calculate the time taken
	duration := time.Since(startTime)
	fmt.Println("Query execution time:", duration)

	// Log the response from InfluxDB
	log.Printf("Received response from InfluxDB: %s\n", resp.Status)

	// Copy the response to the client
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func forwardRequest(r *http.Request) (*http.Response, error) {
	// Create a new request to forward to InfluxDB
	influxReq, err := http.NewRequest(r.Method, fmt.Sprintf("%s:%s", influxDBServer, r.URL.Path), r.Body)
	if err != nil {
		return nil, err
	}
	influxReq.Header = r.Header

	// Send the request to InfluxDB
	client := &http.Client{}
	return client.Do(influxReq)
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
