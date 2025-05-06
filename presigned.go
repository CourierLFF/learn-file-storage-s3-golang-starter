package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedUrl(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	gottenObject, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", fmt.Errorf("could not get presign object: %v", err)
	}

	return gottenObject.URL, nil

}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	returnedVideo := video

	if returnedVideo.VideoURL == nil {
		fmt.Printf("Warning: Video with ID %v has a nil URL\n", video.ID)
		return returnedVideo, nil
	}

	splits := strings.Split(*returnedVideo.VideoURL, ",")
	if len(splits) != 2 {
		fmt.Printf("Warning: Video with ID %v has an old-format URL: %s\n", video.ID, *returnedVideo.VideoURL)
		return returnedVideo, nil
	}

	presignedUrl, err := generatePresignedUrl(cfg.s3Client, splits[0], splits[1], time.Hour)
	if err != nil {
		return database.Video{}, fmt.Errorf("could not presign url: %v", err)
	}

	returnedVideo.VideoURL = &presignedUrl
	return returnedVideo, nil
}
