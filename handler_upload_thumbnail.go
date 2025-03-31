package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get file", err)
		return
	}
	defer file.Close()
	//imgdata, err := io.ReadAll(file)
	//if err != nil {
	//	respondWithError(w, http.StatusBadRequest, "Couldn't read file", err)
	//	return
	//}
	dbvideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get video", err)
		return
	}
	if dbvideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Auth user is not the owner of the video", err)
		return
	}
	//vidthumb := thumbnail{data: imgdata, mediaType: header.Header.Get("Content-Type")}

	//imgencoded := base64.StdEncoding.EncodeToString(imgdata)
	//dataurl := fmt.Sprintf("data:%s;base64,%s", header.Header.Get("Content-Type"), imgdata)
	filehead, _, _ := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if filehead != "image/jpeg" && filehead != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	}
	var filext string
	switch filehead {
	case "image/jpeg":
		filext = ".jpg"
	case "image/png":
		filext = ".png"
	}
	randbytes := make([]byte, 32)
	rand.Read(randbytes)
	randName := base64.RawURLEncoding.EncodeToString(randbytes)
	videoSaveName := fmt.Sprintf("%s%s", randName, filext)
	thumbfile := filepath.Join(cfg.assetsRoot, videoSaveName)
	asd, err := os.Create(thumbfile)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create file", err)
		return
	}
	_, err = io.Copy(asd, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't copy file", err)
		return
	}
	thumblink := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, videoSaveName)
	dbvideo.ThumbnailURL = &thumblink
	err = cfg.db.UpdateVideo(dbvideo)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't update video", err)
		return
	}
	dbvideojson, _ := json.Marshal(dbvideo)
	respondWithJSON(w, http.StatusOK, dbvideojson)
}
