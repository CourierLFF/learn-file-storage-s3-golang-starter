package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {

	type Stream struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	type VideoData struct {
		Streams []Stream `json:"streams"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run ffprobe: %v", err)
	}

	var videoData VideoData
	if err := json.Unmarshal(outBuf.Bytes(), &videoData); err != nil {
		return "", fmt.Errorf("ERROR! Could not unmarshal command data %v", err)
	}

	if len(videoData.Streams) == 0 {
		return "", fmt.Errorf("no streams found in video")
	}

	width := videoData.Streams[0].Width
	height := videoData.Streams[0].Height

	aspectRatio := float64(width) / float64(height)

	// 16:9 ≈ 1.777...
	if aspectRatio > 1.7 && aspectRatio < 1.8 {
		return "landscape", nil
		// 9:16 ≈ 0.5625
	} else if aspectRatio > 0.55 && aspectRatio < 0.57 {
		return "portrait", nil
	} else {
		return "other", nil
	}
}
