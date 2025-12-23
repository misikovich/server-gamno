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
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fred1268/go-clap/clap"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

type config struct {
	Migrate bool `clap:"--migrate,-m"`
	ClearDB bool `clap:"--YES-I-REALLY-WANT-TO-DELETE-ALL-DATA"`
	Update  bool `clap:"--update,-u"`
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
	LogoURL         string `json:"logo_url"`
}

type YouTubeResponse struct {
	Items []struct {
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
			ChannelID    string `json:"channelId"`
		} `json:"snippet"`
		Status struct {
			Embeddable bool `json:"embeddable"`
		} `json:"status"`
		ContentDetails struct {
			ContentRating struct {
				YTRating string `json:"ytRating"`
			} `json:"contentRating"`
		} `json:"contentDetails"`
	} `json:"items"`
}

type YTChannelResponse struct {
	Items []struct {
		Snippet struct {
			Thumbnails struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
			} `json:"thumbnails"`
		} `json:"snippet"`
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

	logo, err := fetchYTLogoLink(video.ChannelID)
	if err != nil {
		log.Println("Error fetching logo: ", err)
		logo = ""
	}

	//return json response in VideoResponse format
	w.Header().Set("Content-Type", "application/json")
	response := VideoResponse{
		ID:              video.ID,
		VideoName:       video.VideoName,
		VideoAuthorName: video.VideoAuthorName,
		IsEmbeddable:    video.IsEmbeddable,
		LogoURL:         logo,
	}
	json.NewEncoder(w).Encode(response)
	log.Println("Requested random video: " + video.ID)
}

func handleRandom(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// if len(videos) == 0 {
	// 	http.Error(w, "No videos available", http.StatusNotFound)
	// 	log.Println("Request for random video failed, no videos available")
	// 	return
	// }

	randomVideo := getRandomVideo(videos)
	fmt.Fprintln(w, randomVideo)
	log.Println("Requested random video: " + randomVideo)
}

func fetchYTVideoInfo(id string) (YouTubeResponse, error) {
	apiKey := env.YTDataAPIv3Key.Get()
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?id=%s&key=%s&part=snippet,status,contentDetails", id, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("[yt] Error fetching video info: ", err)
		return YouTubeResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("[yt] Error fetching video info: ", resp.Status)
		return YouTubeResponse{}, err
	}

	var ytResp YouTubeResponse
	if err := json.NewDecoder(resp.Body).Decode(&ytResp); err != nil {
		log.Println("[yt] Error parsing YouTube response: ", err)
		return YouTubeResponse{}, err
	}

	if len(ytResp.Items) == 0 {
		log.Println("[yt] Video not found: ", id)
		return YouTubeResponse{}, errors.New("video not found")
	}
	return ytResp, nil
}

func assembleVideo(ytResp YouTubeResponse, ip string, id string) db.Video {
	item := ytResp.Items[0]

	var embeddable bool = item.Status.Embeddable && item.ContentDetails.ContentRating.YTRating == ""

	return db.Video{
		ID:              id,
		VideoName:       item.Snippet.Title,
		VideoAuthorName: item.Snippet.ChannelTitle,
		IsEmbeddable:    embeddable,
		AddedAt:         time.Now().Unix(),
		AddedFromIP:     ip,
		ChannelID:       item.Snippet.ChannelID,
	}
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

	sanitizedID, err := SANITIZE_ID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Request for adding video failed, [invalid video 'id' parameter]")
		log.Println("TRY: ", id)
		return
	}
	id = sanitizedID

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

	ip := r.RemoteAddr
	if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
		ip = prior
	}

	ytResp, err := fetchYTVideoInfo(id)
	if err != nil {
		log.Println("Error fetching video info: ", err)
		http.Error(w, "Failed to fetch video info", http.StatusInternalServerError)
		return
	}

	video := assembleVideo(ytResp, ip, id)

	log.Printf("Parsed video:\n- ID: %s\n- Name: %s\n- Author: %s\n- Embeddable: %t\n- Timestamp: %d\n- IP: %s\n Channel ID: %s\n",
		video.ID,
		video.VideoName,
		video.VideoAuthorName,
		video.IsEmbeddable,
		video.AddedAt,
		video.AddedFromIP,
		video.ChannelID,
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
		LogoURL:         "",
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
		ytResp, err := fetchYTVideoInfo(video)
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

func fetchYTLogoLink(channelId string) (string, error) {
	apiKey := env.YTDataAPIv3Key.Get()
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/channels?part=snippet&id=%s&key=%s", channelId, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ytResp YTChannelResponse
	if err := json.NewDecoder(resp.Body).Decode(&ytResp); err != nil {
		return "", err
	}

	if len(ytResp.Items) == 0 {
		return "", errors.New("channel not found")
	}
	return ytResp.Items[0].Snippet.Thumbnails.Default.URL, nil
}

func SANITIZE_ID(id string) (string, error) {
	if len(id) != 11 {
		return "", errors.New("sybau nigga")
	}
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	if re.MatchString(id) {
		return "", errors.New("sybau nigga")
	}
	return id, nil
}

func main() {
	args := parseArgs()
	Env()
	db.InitDB()

	log.Println("Migrate:", args.Migrate)
	log.Println("ClearDB:", args.ClearDB)
	log.Println("Update:", args.Update)
	if args.Migrate {
		migrateDBfromJSON()
	}
	if args.ClearDB {
		db.ClearDB()
	}
	if args.Update {
		videos, err := db.GetAllVideos()
		if err != nil {
			log.Println("Error getting videos:", err)
			return
		}
		for _, video := range videos {
			log.Println(video, "- updating video credentials...")
			ytResp, err := fetchYTVideoInfo(video.ID)
			if err != nil {
				log.Println("Error fetching video info: ", err)
				continue
			}
			updatedVideo := assembleVideo(ytResp, video.AddedFromIP, video.ID)
			err = db.UpdateVideo(updatedVideo)
			if err != nil {
				log.Println("Error updating video: ", err)
			}
			log.Println(video, "- updated")
		}
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
