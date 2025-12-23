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
	ChannelID       string `json:"channel_id"`
}

var DB *sql.DB

func InitDB() {
	db, err := sql.Open("sqlite3", env.DBPath.Get())
	if err != nil {
		log.Fatal("[db] Error opening database: ", err)
	}
	log.Println("[db] Database opened successfully: " + env.DBPath.Get())

	// id (text),
	// video_name (text),
	// video_author_username (text),
	// is_embeddable (bool),
	// added_at (unix timestamp),
	// added_from_ip (ip address)
	// channel_id (text)
	sqlStmt := "CREATE TABLE IF NOT EXISTS videos (id TEXT PRIMARY KEY, video_name TEXT, video_author_username TEXT, is_embeddable BOOLEAN, added_at INTEGER, added_from_ip TEXT, channel_id TEXT)"
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal("[db] Error creating table: ", err, sqlStmt)
	}
	log.Println("[db] Table created successfully")

	DB = db
}

func GetDB() *sql.DB {
	return DB
}

// does it handle duplicates?
// answer: no
// solution: use INSERT OR IGNORE
func InsertVideo(video Video) error {
	stmt, err := DB.Prepare("INSERT OR IGNORE INTO videos (id, video_name, video_author_username, is_embeddable, added_at, added_from_ip, channel_id) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("[db] Error preparing statement: ", err)
		return err
	}
	_, err = stmt.Exec(video.ID, video.VideoName, video.VideoAuthorName, video.IsEmbeddable, video.AddedAt, video.AddedFromIP, video.ChannelID)
	if err != nil {
		log.Println("[db] Error inserting video: ", err)
		return err
	}
	log.Println("[db] Video inserted successfully")
	return nil
}

func GetRandomVideo() (Video, error) {
	stmt, err := DB.Prepare("SELECT * FROM videos ORDER BY RANDOM() LIMIT 1")
	if err != nil {
		log.Println("[db] Error preparing statement: ", err)
		return Video{}, err
	}
	defer stmt.Close()

	var video Video
	err = stmt.QueryRow().Scan(&video.ID, &video.VideoName, &video.VideoAuthorName, &video.IsEmbeddable, &video.AddedAt, &video.AddedFromIP, &video.ChannelID)
	if err != nil {
		log.Println("[db] Error getting random video: ", err)
		return Video{}, err
	}
	return video, nil
}

func GetVideosByIP(ip string) ([]Video, error) {
	stmt, err := DB.Prepare("SELECT * FROM videos WHERE added_from_ip = ?")
	if err != nil {
		log.Println("[db] Error preparing statement: ", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(ip)
	if err != nil {
		log.Println("[db] Error getting videos by IP: ", err)
		return nil, err
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		var video Video
		err = rows.Scan(&video.ID, &video.VideoName, &video.VideoAuthorName, &video.IsEmbeddable, &video.AddedAt, &video.AddedFromIP, &video.ChannelID)
		if err != nil {
			log.Println("[db] Error scanning row: ", err)
		}
		videos = append(videos, video)
	}
	return videos, nil
}

func GetAllVideos() ([]Video, error) {
	//sort by added_at oldest first (asc)
	stmt, err := DB.Prepare("SELECT * FROM videos ORDER BY added_at ASC")
	if err != nil {
		log.Println("[db] Error preparing statement: ", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		log.Println("[db] Error getting videos: ", err)
		return nil, err
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		var video Video
		err = rows.Scan(&video.ID, &video.VideoName, &video.VideoAuthorName, &video.IsEmbeddable, &video.AddedAt, &video.AddedFromIP, &video.ChannelID)
		if err != nil {
			log.Println("Error scanning row: ", err)
		}
		videos = append(videos, video)
	}
	return videos, nil
}

func CountSavedVideos() (int, error) {
	stmt, err := DB.Prepare("SELECT COUNT(*) FROM videos")
	if err != nil {
		log.Println("[db] Error preparing statement: ", err)
		return 0, err
	}
	defer stmt.Close()

	var count int
	err = stmt.QueryRow().Scan(&count)
	if err != nil {
		log.Println("[db] Error getting number of videos: ", err)
		return 0, err
	}
	return count, nil
}

func IsVideoSaved(id string) (bool, error) {
	stmt, err := DB.Prepare("SELECT * FROM videos WHERE id = ?")
	if err != nil {
		log.Println("[db] Error preparing statement: ", err)
		return false, err
	}
	defer stmt.Close()

	var video Video
	err = stmt.QueryRow(id).Scan(&video.ID, &video.VideoName, &video.VideoAuthorName, &video.IsEmbeddable, &video.AddedAt, &video.AddedFromIP, &video.ChannelID)
	if err != nil {
		// log.Println("[db] Video not found: ", err)
		return false, nil
	}
	return true, nil
}

func ClearDB() {
	stmt, err := DB.Prepare("DELETE FROM videos")
	if err != nil {
		log.Println("[db] Error preparing statement: ", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		log.Println("[db] Error clearing database: ", err)
		return
	}
	log.Println("[db] Database cleared successfully")
}

func UpdateVideo(video Video) error {
	stmt, err := DB.Prepare("UPDATE videos SET video_name = ?, video_author_username = ?, is_embeddable = ?, added_at = ?, added_from_ip = ?, channel_id = ? WHERE id = ?")
	if err != nil {
		log.Println("[db] Error preparing statement: ", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(video.VideoName, video.VideoAuthorName, video.IsEmbeddable, video.AddedAt, video.AddedFromIP, video.ChannelID, video.ID)
	if err != nil {
		log.Println("[db] Error updating video: ", err)
		return err
	}
	log.Println("[db] Video updated successfully")
	return nil
}
