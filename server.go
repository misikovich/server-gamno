package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go3/db"
	"go3/env"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fred1268/go-clap/clap"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

type config struct {
	Migrate bool `clap:"--migrate,-m"`
	ClearDB bool `clap:"--clear-db,-c"`
}

var (
	videos []string
	mu     sync.Mutex
)

type VideoResponse struct {
	ID              string `json:"id"`
	VideoName       string `json:"video_name"`
	VideoAuthorName string `json:"video_author_name"`
	IsEmbeddable    bool   `json:"is_embeddable"`
}

type YouTubeResponse struct {
	Items []struct {
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
		} `json:"snippet"`
		Status struct {
			Embeddable bool `json:"embeddable"`
		} `json:"status"`
	} `json:"items"`
}

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

// func addVideo(videos []string, newID string) ([]string, error) {
// 	for _, id := range videos {
// 		if id == newID {
// 			log.Println("Video ID already exists in the list, skipping: " + newID)
// 			return videos, errors.New("video id already exists")
// 		}
// 	}
// 	if !isValidVideoID(newID) {
// 		log.Println("Invalid video ID: " + newID)
// 		return videos, errors.New("invalid video id")
// 	}
// 	videos = append(videos, newID)
// 	log.Println("Added new video ID: " + newID)
// 	filename := env.VideosIDFile.Get()
// 	err := SaveVideos(filename, videos)
// 	if err != nil {
// 		log.Fatal("Error saving videos: \n" + err.Error())
// 	}
// 	return videos, nil
// }

func getRandomVideo(videos []string) string {
	return videos[rand.Intn(len(videos))]
}

// func isValidVideoID(id string) bool {
// 	if len(id) != 11 {
// 		return false
// 	}
// 	url := fmt.Sprintf(
// 		"https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=%s&format=json",
// 		id,
// 	)
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		log.Println("Error getting video info: \n" + err.Error())
// 		return false
// 	}
// 	defer resp.Body.Close()
// 	return resp.StatusCode == http.StatusOK
// }

func handleRandomV2(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	video, err := db.GetRandomVideo()
	if err != nil {
		http.Error(w, "Failed to get random video", http.StatusInternalServerError)
		log.Println("Error getting random video: ", err)
		return
	}

	//return json response in VideoResponse format
	w.Header().Set("Content-Type", "application/json")
	response := VideoResponse{
		ID:              video.ID,
		VideoName:       video.VideoName,
		VideoAuthorName: video.VideoAuthorName,
		IsEmbeddable:    video.IsEmbeddable,
	}
	json.NewEncoder(w).Encode(response)
	log.Println("Requested random video: " + video.ID)
}

func handleRandom(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if len(videos) == 0 {
		http.Error(w, "No videos available", http.StatusNotFound)
		log.Println("Request for random video failed, no videos available")
		return
	}

	randomVideo := getRandomVideo(videos)
	fmt.Fprintln(w, randomVideo)
	log.Println("Requested random video: " + randomVideo)
}

func getYTvideoInfo(id string, w http.ResponseWriter) (YouTubeResponse, error) {
	apiKey := env.YTDataAPIv3Key.Get()
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?id=%s&key=%s&part=snippet,status", id, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to fetch video info", http.StatusInternalServerError)
		log.Println("[yt] Error fetching video info: ", err)
		return YouTubeResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "YouTube API returned error", resp.StatusCode)
		log.Println("[yt] Error fetching video info: ", resp.Status)
		return YouTubeResponse{}, err
	}

	var ytResp YouTubeResponse
	if err := json.NewDecoder(resp.Body).Decode(&ytResp); err != nil {
		http.Error(w, "Failed to parse YouTube response", http.StatusInternalServerError)
		log.Println("[yt] Error parsing YouTube response: ", err)
		return YouTubeResponse{}, err
	}

	if len(ytResp.Items) == 0 {
		http.Error(w, "Video not found", http.StatusNotFound)
		log.Println("[yt] Video not found: ", id)
		return YouTubeResponse{}, errors.New("video not found")
	}
	return ytResp, nil
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		log.Println("Request for adding video failed, [method not allowed]")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing video 'id' parameter", http.StatusBadRequest)
		log.Println("Request for adding video failed, [missing video 'id' parameter]")
		return
	}

	log.Println("Requested adding video: " + id)

	//check if video exists
	exists, err := db.IsVideoSaved(id)
	if err != nil {
		http.Error(w, "Failed to check if video exists", http.StatusInternalServerError)
		log.Println("Error checking if video exists: ", err)
		return
	}
	if exists {
		http.Error(w, "Video already exists", http.StatusConflict)
		log.Println("Request for adding video failed, [video already exists]")
		return
	}

	ytResp, err := getYTvideoInfo(id, w)
	if err != nil {
		log.Println("Error fetching video info:", err)
		return
	}

	item := ytResp.Items[0]

	ip := r.RemoteAddr
	if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
		ip = prior
	}

	video := db.Video{
		ID:              id,
		VideoName:       item.Snippet.Title,
		VideoAuthorName: item.Snippet.ChannelTitle,
		IsEmbeddable:    item.Status.Embeddable,
		AddedAt:         time.Now().Unix(),
		AddedFromIP:     ip,
	}

	log.Printf("Parsed video:\n- ID: %s\n- Name: %s\n- Author: %s\n- Embeddable: %t\n- Timestamp: %d\n- IP: %s\n",
		video.ID,
		video.VideoName,
		video.VideoAuthorName,
		video.IsEmbeddable,
		video.AddedAt,
		video.AddedFromIP,
	)

	// Insert into Database
	if err := db.InsertVideo(video); err != nil {
		log.Println("Failed to insert video into database: ", err)
		// We continue even if DB insert fails? Or return error?
		// For now, let's just log it and continue with the JSON file update
		http.Error(w, "servaku pizda", http.StatusInternalServerError)
		return
	}

	// mu.Lock()
	// defer mu.Unlock()

	//return json response in VideoResponse format
	w.Header().Set("Content-Type", "application/json")
	response := VideoResponse{
		ID:              video.ID,
		VideoName:       video.VideoName,
		VideoAuthorName: video.VideoAuthorName,
		IsEmbeddable:    video.IsEmbeddable,
	}
	json.NewEncoder(w).Encode(response)
	//fmt.Fprintf(w, "Successfully added video '%s' (%s)\n", video.ID, video.VideoName)
}

func migrateDBfromJSON() {
	videos := loadVideos()
	for _, video := range videos {
		log.Println(video, "- migrating video...")
		exists, err := db.IsVideoSaved(video)
		if err != nil {
			log.Println("Error checking if video exists: ", err)
			return
		}
		if exists {
			log.Println(video, "- video already exists")
			continue
		}
		ytResp, err := getYTvideoInfo(video, nil)
		if err != nil {
			log.Println("Error fetching video info: ", err)
			return
		}
		err = db.InsertVideo(db.Video{
			ID:              video,
			VideoName:       ytResp.Items[0].Snippet.Title,
			VideoAuthorName: ytResp.Items[0].Snippet.ChannelTitle,
			IsEmbeddable:    ytResp.Items[0].Status.Embeddable,
			AddedAt:         time.Now().Unix(),
			AddedFromIP:     "migrated",
		})
		if err != nil {
			log.Println("Error inserting video: ", err)
		}
		log.Println(video, "- migrated")
	}
}

func parseArgs() config {
	args := os.Args[1:]
	cfg := config{Migrate: false}

	_, err := clap.Parse(args, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

func main() {
	args := parseArgs()
	Env()
	db.InitDB()

	log.Println("Migrate:", args.Migrate)
	if args.Migrate {
		migrateDBfromJSON()
	}
	if args.ClearDB {
		db.ClearDB()
	}

	count, err := db.CountSavedVideos()
	if err != nil {
		log.Println("Error getting number of videos:", err)
		return
	}
	log.Println("Number of videos:", count)

	mux := http.NewServeMux()
	mux.HandleFunc("/get_random", handleRandom)
	mux.HandleFunc("/v2/get_random", handleRandomV2)
	mux.HandleFunc("/v2/add", handleAdd)

	address := fmt.Sprintf("%s:%s", env.Host.Get(), env.Port.Get())

	if env.UseTLS.Get() == "FALSE" {
		err := serve(address, mux)
		if err != nil {
			log.Fatal("Server failed: ", err)
		}
		return
	}
	if env.UseTLS.Get() == "TRUE" {
		err := serveTLS(address, mux)
		if err != nil {
			log.Fatal("Server failed: ", err)
		}
		return
	}
}

func serve(addr string, mux *http.ServeMux) error {
	log.Printf("Server starting on http://%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func serveTLS(addr string, mux *http.ServeMux) error {
	allowedOrigins := strings.Split(env.AllowedOrigins.Get(), ",")
	allowedMethods := strings.Split(env.AllowedMethods.Get(), ",")
	log.Println("AllowedOrigins: ", allowedOrigins)
	log.Println("AllowedMethods: ", allowedMethods)
	handler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(mux)

	log.Println("TLS cert path: ", env.TLSCertPath.Get())
	log.Println("TLS key path: ", env.TLSKeyPath.Get())
	log.Printf("Server starting on https://%s\n", addr)
	return http.ListenAndServeTLS(addr, env.TLSCertPath.Get(), env.TLSKeyPath.Get(), handler)
}
