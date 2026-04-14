package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"html/template"
	_ "modernc.org/sqlite"
)

var (
	db 			*sql.DB
	uploadDir	string
	appURL		string
)

func initDatabase(path string) error {
	var err error
	db, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS files (
			id TEXT PRIMARY KEY,
			name TEXT,
			extension TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		)`,
	)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func generateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.RawURLEncoding.EncodeToString(b), err
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
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uploadedFile, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer uploadedFile.Close()

	id, err := generateRandomString(12)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(header.Filename)
	ext := filepath.Ext(filename)
	name := filename[:len(filename) - len(ext)]

	now := time.Now()
	expiredAt := now.Add(24 * time.Hour)

	_, err = db.ExecContext(
		context.Background(),
		`INSERT INTO files (id, name, extension, expires_at) VALUES (?, ?, ?, ?)`, id, name, ext, expiredAt,
	)

	path := filepath.Join(uploadDir, id + ext)
	
	targetFile, err := os.Create(path)
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

	t, err := template.ParseFiles("template.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	downloadUrl := fmt.Sprintf("%s/d/%s", appURL, id)

	data := struct {
		FileName string
		FileID string
		DownloadURL string
	}{
		FileName: filename,
		FileID: id,
		DownloadURL: downloadUrl,
	}
	
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusCreated)
	t.Execute(w, data)
}

func fileDownloadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/d/"):]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var name, ext string

	row := db.QueryRowContext(
		context.Background(),
		`SELECT name, extension FROM files WHERE id=?`, id,
	)

	err := row.Scan(&name, &ext)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	filename := name + ext
	contentDisposition := fmt.Sprintf("attachment; filename=\"%s\"", filename)
	w.Header().Set("Content-Disposition", contentDisposition)

	filepath := filepath.Join(uploadDir, id + ext)
	http.ServeFile(w, r, filepath)
}

func removeExpiredFiles() {
	now := time.Now()

	rows, err := db.QueryContext(
		context.Background(),
		`SELECT id, extension FROM files WHERE expires_at <= ?`, now,
	)
	if err != nil {
		log.Println(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, ext string
		if err := rows.Scan(&id, &ext); err != nil {
			continue
		}

		path := filepath.Join(uploadDir, id + ext)
		if err := os.Remove(path); err != nil {
			log.Println(err)
		} else {
			log.Printf("Deleted expired file: %s", path)
		}
	}

	_, err = db.ExecContext(
		context.Background(),
		`DELETE FROM files WHERE expires_at <= ?`, now,
	)
	if err != nil {
		log.Println(err)
		return
	}
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	initDatabase(dbPath)
	defer db.Close()

	addr := ":8080"
	port := os.Getenv("APP_PORT")
	if port == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr: addr,
	}

	appURL = os.Getenv("APP_URL")
	uploadDir = os.Getenv("UPLOAD_DIR")

	go func(){
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			removeExpiredFiles()
		}
	}()
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/upload", fileUploadHandler)
	http.HandleFunc("/d/", fileDownloadHandler)

	shutdownChan := make(chan bool, 1)

	go func(){
		log.Println("Starting server on :8080.")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}

		shutdownChan <- true
		log.Println("Stopped serving new connections.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}

	<-shutdownChan
	log.Println("Graceful shutdown complete.")
}