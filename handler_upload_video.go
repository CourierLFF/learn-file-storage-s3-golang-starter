package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not get video metadata", err)
		return
	}

	if metadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not owner of video", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	defer file.Close()

	contentType := header.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not parse media type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid filetype", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create temporary file", err)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	io.Copy(tempFile, file)

	tempFile.Seek(0, io.SeekStart)

	randomBytes := make([]byte, 16)
	_, err = rand.Read(randomBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not generate random key", err)
		return
	}

	aspect_ratio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not get video aspect ratio", err)
		return
	}

	processedVideoPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not process video", err)
		return
	}

	processedVideo, err := os.Open(processedVideoPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not open video", err)
		return
	}
	defer processedVideo.Close()

	keyString := fmt.Sprintf("%v/%x.mp4", aspect_ratio, randomBytes) // Convert to hex string and add extension

	putOutput, err := cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &keyString,
		Body:        processedVideo,
		ContentType: &mediaType,
	})
	if err != nil {
		fmt.Printf("%v\n", putOutput)
		respondWithError(w, http.StatusInternalServerError, "Could not upload to S3", err)
		return
	}

	//videoUrl := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, keyString)
	videoUrl := fmt.Sprintf("%v/%v", cfg.s3CfDistribution, keyString)

	metadata.VideoURL = &videoUrl
	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video in database", err)
		return
	}

}
