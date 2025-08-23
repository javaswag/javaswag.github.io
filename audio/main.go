package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed templates/*
var resources embed.FS

var t = template.Must(template.ParseFS(resources, "templates/*"))

const maxBodySize = 500 * 1024 * 1024 // 500 MB in bytes

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	// Now proceed with parsing the multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		// http.MaxBytesReader will return an error here if the size is exceeded
		http.Error(w, "Request entity is too large.", http.StatusRequestEntityTooLarge)
		return
	}

	// Get the file from form data
	file, header, err := r.FormFile("audioFile")
	if err != nil {
		http.Error(w, "Unable to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".mp3" {
		http.Error(w, "Only MP3 files are allowed", http.StatusBadRequest)
		return
	}

	// Validate file size (500MB max)
	if header.Size > 500<<20 {
		http.Error(w, "File size exceeds 500MB limit", http.StatusBadRequest)
		return
	}

	// Create uploads directory if it doesn't exist
	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		http.Error(w, "Unable to create uploads directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the destination file
	dst, err := os.Create(filepath.Join(uploadsDir, header.Filename))
	if err != nil {
		http.Error(w, "Unable to create file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to destination
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Unable to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Success response
	data := map[string]interface{}{
		"Filename": header.Filename,
		"SizeMB":   fmt.Sprintf("%.2f", float64(header.Size)/(1024*1024)),
	}

	t.ExecuteTemplate(w, "success.html.tmpl", data)

	log.Printf("File uploaded successfully: %s (%.2f MB)", header.Filename, float64(header.Size)/(1024*1024))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"Region": os.Getenv("FLY_REGION"),
		}

		t.ExecuteTemplate(w, "index.html.tmpl", data)
	})

	http.HandleFunc("/upload", handleFileUpload)

	log.Printf("listening on http://localhost:%s\n", port)
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      nil,
		ReadTimeout:  200 * time.Second,
		WriteTimeout: 200 * time.Second,
		IdleTimeout:  200 * time.Second,
	}
	server.ListenAndServe()
	// log.Fatal(http.ListenAndServe(":"+port, nil))
}
