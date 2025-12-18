package main

import (
	"errors"
	"fmt"
	"go3/env"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"

	"github.com/rs/cors"
)

var (
	videos []string
	mu     sync.Mutex
)

func Env() {
	env.LoadEnv()
	fmt.Println("Hello, World!")
	fmt.Println("Host: " + env.Host.Get())
	fmt.Println("Port: " + env.Port.Get())
	fmt.Println("API Host: " + env.APIHost.Get())
	fmt.Println("API Port: " + env.APIPort.Get())
	fmt.Println("Dev Mode: " + env.DevMode.Get())
	fmt.Println("Videos ID File: " + env.VideosIDFile.Get())
}

// returns videos list
func loadVideos() []string {
	filename := env.VideosIDFile.Get()
	videos, err := LoadVideos(filename)
	if err != nil {
		log.Fatal("Error loading videos: \n" + err.Error())
	}
	log.Printf("Loaded [%d] videos\n", len(videos))
	return videos
}

func addVideo(videos []string, newID string) ([]string, error) {
	for _, id := range videos {
		if id == newID {
			log.Println("Video ID already exists in the list, skipping: " + newID)
			return videos, errors.New("video id already exists")
		}
	}
	if !isValidVideoID(newID) {
		log.Println("Invalid video ID: " + newID)
		return videos, errors.New("invalid video id")
	}
	videos = append(videos, newID)
	log.Println("Added new video ID: " + newID)
	filename := env.VideosIDFile.Get()
	err := SaveVideos(filename, videos)
	if err != nil {
		log.Fatal("Error saving videos: \n" + err.Error())
	}
	return videos, nil
}

func getRandomVideo(videos []string) string {
	return videos[rand.Intn(len(videos))]
}

func isValidVideoID(id string) bool {
	if len(id) != 11 {
		return false
	}
	url := fmt.Sprintf(
		"https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=%s&format=json",
		id,
	)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error getting video info: \n" + err.Error())
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func handleRandom(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if len(videos) == 0 {
		http.Error(w, "No videos available", http.StatusNotFound)
		return
	}

	randomVideo := getRandomVideo(videos)
	fmt.Fprintln(w, randomVideo)
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing video 'id' parameter", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	var err error
	videos, err = addVideo(videos, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Video '%s' added successfully\n", id)
}

func main() {
	Env()

	// Initial load
	videos = loadVideos()

	http.HandleFunc("/get_random", handleRandom)
	http.HandleFunc("/add", handleAdd)

	address := fmt.Sprintf("%s:%s", env.Host.Get(), env.Port.Get())

	if env.UseTLS.Get() == "FALSE" {
		err := serve(address)
		if err != nil {
			log.Fatal("Server failed: ", err)
		}
		return
	}
	if env.UseTLS.Get() == "TRUE" {
		err := serveTLS(address)
		if err != nil {
			log.Fatal("Server failed: ", err)
		}
		return
	}
}

func serve(addr string) error {
	log.Printf("Server starting on http://%s\n", addr)
	return http.ListenAndServe(addr, nil)
}

func serveTLS(addr string) error {
	allowedOrigins := strings.Split(env.AllowedOrigins.Get(), ",")
	allowedMethods := strings.Split(env.AllowedMethods.Get(), ",")
	log.Println("AllowedOrigins: ", allowedOrigins)
	log.Println("AllowedMethods: ", allowedMethods)
	handler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(nil)

	log.Println("TLS cert path: ", env.TLSCertPath.Get())
	log.Println("TLS key path: ", env.TLSKeyPath.Get())
	log.Printf("Server starting on https://%s\n", addr)
	return http.ListenAndServeTLS(addr, env.TLSCertPath.Get(), env.TLSKeyPath.Get(), handler)
}
