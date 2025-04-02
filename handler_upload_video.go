package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	http.MaxBytesReader(w, r.Body, 1<<30)
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

	fmt.Println("uploading video", videoID, "by user", userID)

	dbvideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get video", err)
		return
	}
	if dbvideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Auth user is not the owner of the video", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get file", err)
		return
	}
	defer file.Close()
	filehead, _, _ := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if filehead != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	}
	tempfile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}
	defer os.Remove(tempfile.Name())
	defer tempfile.Close()
	_, err = io.Copy(tempfile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy file", err)
		return
	}
	tempfile.Seek(0, io.SeekStart)
	randbytes := make([]byte, 32)
	rand.Read(randbytes)
	randName := base64.RawURLEncoding.EncodeToString(randbytes) + ".mp4"
	_, err = cfg.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &randName,
		Body:        tempfile,
		ContentType: &filehead,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload file", err)
		return
	}
	videolink := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, randName)
	dbvideo.VideoURL = &videolink
	err = cfg.db.UpdateVideo(dbvideo)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't update video", err)
		return
	}
	dbvideojson, _ := json.Marshal(dbvideo)
	respondWithJSON(w, http.StatusOK, dbvideojson)
}
