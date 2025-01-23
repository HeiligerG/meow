package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/redis/go-redis/v9"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/patrickbucher/meow"
)

type Config map[string]*meow.Endpoint

type ConcurrentConfig struct {
	mu     sync.RWMutex
	config Config
}

var cfg ConcurrentConfig

func main() {
	addr := flag.String("addr", "0.0.0.0", "listen to address")
	port := flag.Uint("port", 8000, "listen on port")
	file := flag.String("file", "config.csv", "CSV file to store the configuration")
	flag.Parse()

	log.SetOutput(os.Stderr)

	cfg.config = mustReadConfig(*file)

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6380"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	http.HandleFunc("/endpoints/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getEndpoint(w, r, rdb)
		case http.MethodPost:
			postEndpoint(w, r, rdb, *file)
		case http.MethodDelete:
			deleteEndpoint(w, r, rdb)
		default:
			log.Printf("request from %s rejected: method %s not allowed",
				r.RemoteAddr, r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/endpoints", func(w http.ResponseWriter, r *http.Request) {
		getEndpoints(w, r, rdb)
	})

	listenTo := fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("listen to %s", listenTo)
	http.ListenAndServe(listenTo, nil)
}

func getEndpoint(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	log.Printf("GET %s from %s", r.URL, r.RemoteAddr)
	identifier, err := extractEndpointIdentifier(r.URL.String())
	if err != nil {
		log.Printf("extract endpoint identifier of %s: %v", r.URL, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cfg.mu.RLock()
	endpoint, ok := cfg.config[identifier]
	cfg.mu.RUnlock()
	if ok {
		payload, err := endpoint.JSON()
		if err != nil {
			log.Printf("convert %v to JSON: %v", endpoint, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(payload)
	} else {
		log.Printf(`no such endpoint "%s"`, identifier)
		w.WriteHeader(http.StatusNotFound)
	}
}

func postEndpoint(w http.ResponseWriter, r *http.Request, rdb *redis.Client, file string) {
	log.Printf("POST %s from %s", r.URL, r.RemoteAddr)
	buf := bytes.NewBufferString("")
	io.Copy(buf, r.Body)
	defer r.Body.Close()
	endpoint, err := meow.EndpointFromJSON(buf.String())
	if err != nil {
		log.Printf("parse JSON body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cfg.mu.RLock()
	_, exists := cfg.config[endpoint.Identifier]
	cfg.mu.RUnlock()
	var status int
	if exists {
		identifierPathParam, err := extractEndpointIdentifier(r.URL.String())
		if err != nil {
			log.Printf("extract endpoint identifier of %s: %v", r.URL, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if identifierPathParam != endpoint.Identifier {
			log.Printf("identifier mismatch: (ressource: %s, body: %s)",
				identifierPathParam, endpoint.Identifier)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		status = http.StatusNoContent
	} else {
		status = http.StatusCreated
	}
	cfg.mu.Lock()
	cfg.config[endpoint.Identifier] = endpoint
	if err := writeConfig(cfg.config, file); err != nil {
		status = http.StatusInternalServerError
	}
	cfg.mu.Unlock()
	w.WriteHeader(status)
}

func deleteEndpoint(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	log.Printf("DELETE %s from %s", r.URL, r.RemoteAddr)
	identifier, err := extractEndpointIdentifier(r.URL.String())
	if err != nil {
		log.Printf("extract endpoint identifier of %s: %v", r.URL, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cfg.mu.Lock()
	delete(cfg.config, identifier)
	cfg.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func getEndpoints(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	if r.Method != http.MethodGet {
		log.Printf("request from %s rejected: method %s not allowed",
			r.RemoteAddr, r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	log.Printf("GET %s from %s", r.URL, r.RemoteAddr)
	payloads := make([]meow.EndpointPayload, 0)
	for _, endpoint := range cfg.config {
		payload := meow.EndpointPayload{
			Identifier:   endpoint.Identifier,
			URL:          endpoint.URL.String(),
			Method:       endpoint.Method,
			StatusOnline: endpoint.StatusOnline,
			Frequency:    endpoint.Frequency.String(),
			FailAfter:    endpoint.FailAfter,
		}
		payloads = append(payloads, payload)
	}
	data, err := json.Marshal(payloads)
	if err != nil {
		log.Printf("serialize payloads: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

const endpointIdentifierPatternRaw = "^/endpoints/([a-z][-a-z0-9]+)$"

var endpointIdentifierPattern = regexp.MustCompile(endpointIdentifierPatternRaw)

func extractEndpointIdentifier(endpoint string) (string, error) {
	matches := endpointIdentifierPattern.FindStringSubmatch(endpoint)
	if len(matches) == 0 {
		return "", fmt.Errorf(`endpoint "%s" does not match pattern "%s"`,
			endpoint, endpointIdentifierPatternRaw)
	}
	return matches[1], nil
}

func writeConfig(config Config, configPath string) error {
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf(`open "%s" for write: %v`, configPath, err)
	}

	writer := csv.NewWriter(file)
	defer file.Close()
	for _, endpoint := range config {
		record := []string{
			endpoint.Identifier,
			endpoint.URL.String(),
			endpoint.Method,
			strconv.Itoa(int(endpoint.StatusOnline)),
			endpoint.Frequency.String(),
			strconv.Itoa(int(endpoint.FailAfter)),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf(`write endpoint "%s": %v`, endpoint, err)
		}
	}
	writer.Flush()
	return nil
}

func mustReadConfig(configPath string) Config {
	file, err := os.Open(configPath)
	if os.IsNotExist(err) {
		log.Printf(`the config file "%s" does not exist`, configPath)
		return Config{}
	}

	config := make(Config, 0)
	reader := csv.NewReader(file)
	defer file.Close()
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("the config file '%s' is malformed: %v", configPath, err)
	}
	for i, line := range records {
		endpoint, err := meow.EndpointFromRecord(line)
		if err != nil {
			log.Fatalf(`line %d: "%s": %v`, i, line, err)
		}
		config[endpoint.Identifier] = endpoint
	}
	return config
}
