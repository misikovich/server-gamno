package main

import (
	"encoding/json"
	"os"
)

// VideoList represents a collection of YouTube video IDs
type VideoList struct {
	VideoIDs []string `json:"video_ids"`
}

// SaveVideos saves a list of video IDs to a JSON file
func SaveVideos(filename string, ids []string) error {
	data := VideoList{VideoIDs: ids}
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, file, 0644)
}

// LoadVideos loads a list of video IDs from a JSON file
func LoadVideos(filename string) ([]string, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var data VideoList
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}
	return data.VideoIDs, nil
}
