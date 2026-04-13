package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
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

	w.Write([]byte("File successfully uploaded"))
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/upload", fileUploadHandler)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}