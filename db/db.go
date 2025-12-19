package db

import (
	"database/sql"
	"log"

	"go3/env"
)

type Video struct {
	ID              string `json:"id"`
	VideoName       string `json:"video_name"`
	VideoAuthorName string `json:"video_author_name"`
	IsEmbeddable    bool   `json:"is_embeddable"`
	AddedAt         int64  `json:"added_at"`
	AddedFromIP     string `json:"added_from_ip"`
}

var DB *sql.DB

func InitDB() {
	db, err := sql.Open("sqlite3", env.DBPath.Get())
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}
	log.Println("Database opened successfully: " + env.DBPath.Get())

	// id (text),
	// video_name (text),
	// video_author_username (text),
	// is_embeddable (bool),
	// added_at (unix timestamp),
	// added_from_ip (ip address)
	sqlStmt := "CREATE TABLE IF NOT EXISTS videos (id TEXT PRIMARY KEY, video_name TEXT, video_author_username TEXT, is_embeddable BOOLEAN, added_at INTEGER, added_from_ip TEXT)"
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal("Error creating table: ", err, sqlStmt)
	}
	log.Println("Table created successfully")

	DB = db
}

func GetDB() *sql.DB {
	return DB
}

// does it handle duplicates?
// answer: no
// solution: use INSERT OR IGNORE
func InsertVideo(video Video) error {
	stmt, err := DB.Prepare("INSERT OR IGNORE INTO videos (id, video_name, video_author_username, is_embeddable, added_at, added_from_ip) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("Error preparing statement: ", err)
		return err
	}
	_, err = stmt.Exec(video.ID, video.VideoName, video.VideoAuthorName, video.IsEmbeddable, video.AddedAt, video.AddedFromIP)
	if err != nil {
		log.Println("Error inserting video: ", err)
		return err
	}
	log.Println("Video inserted successfully")
	return nil
}

func GetRandomVideo() (Video, error) {
	stmt, err := DB.Prepare("SELECT * FROM videos ORDER BY RANDOM() LIMIT 1")
	if err != nil {
		log.Println("Error preparing statement: ", err)
		return Video{}, err
	}
	defer stmt.Close()

	var video Video
	err = stmt.QueryRow().Scan(&video.ID, &video.VideoName, &video.VideoAuthorName, &video.IsEmbeddable, &video.AddedAt, &video.AddedFromIP)
	if err != nil {
		log.Println("Error getting random video: ", err)
		return Video{}, err
	}
	return video, nil
}

func GetVideosByIP(ip string) ([]Video, error) {
	stmt, err := DB.Prepare("SELECT * FROM videos WHERE added_from_ip = ?")
	if err != nil {
		log.Println("Error preparing statement: ", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(ip)
	if err != nil {
		log.Println("Error getting videos by IP: ", err)
		return nil, err
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		var video Video
		err = rows.Scan(&video.ID, &video.VideoName, &video.VideoAuthorName, &video.IsEmbeddable, &video.AddedAt, &video.AddedFromIP)
		if err != nil {
			log.Println("Error scanning row: ", err)
		}
		videos = append(videos, video)
	}
	return videos, nil
}

func GetAllVideos() ([]Video, error) {
	stmt, err := DB.Prepare("SELECT * FROM videos")
	if err != nil {
		log.Println("Error preparing statement: ", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		log.Println("Error getting videos: ", err)
		return nil, err
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		var video Video
		err = rows.Scan(&video.ID, &video.VideoName, &video.VideoAuthorName, &video.IsEmbeddable, &video.AddedAt, &video.AddedFromIP)
		if err != nil {
			log.Println("Error scanning row: ", err)
		}
		videos = append(videos, video)
	}
	return videos, nil
}
