package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func fileUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST request is allowed", http.StatusBadRequest)
		return
	}

	// Limit file uploads to 50 MB
	r.Body = http.MaxBytesReader(w, r.Body, 50 << 20)

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uploadedFile, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer uploadedFile.Close()

	filename := header.Filename
	targetFile, err := os.Create("uploads/" + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, uploadedFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("localhost:8080/d/" + filename))
}

func fileDownloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/d/"):]
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}

	contentDisposition := fmt.Sprintf("attachment; filename=\"%s\"", filename)
	w.Header().Set("Content-Disposition", contentDisposition)

	filepath := filepath.Join("uploads", filename)
	http.ServeFile(w, r, filepath)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/upload", fileUploadHandler)
	http.HandleFunc("/d/", fileDownloadHandler)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}