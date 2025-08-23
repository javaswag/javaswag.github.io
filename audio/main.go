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
	"waveform/internal"
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

	go func() {
		generatedFile, err := internal.Run(dst.Name(), uploadsDir)
		if err != nil {
			fmt.Printf("Error generating file: %v\n", err)
		}
		err = os.Remove(dst.Name())
		if err != nil {
			fmt.Printf("Error deleting file: %v\n", err)
		}
		fmt.Printf("Generated file: %s\n", generatedFile)
	}()

	if err != nil {
		http.Error(w, "Unable to generate file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Success response
	data := map[string]interface{}{
		"Region":         os.Getenv("FLY_REGION"),
		"UploadSuccess":  true,
		"UploadFilename": header.Filename,
		"UploadSizeMB":   fmt.Sprintf("%.2f", float64(header.Size)/(1024*1024)),
	}

	t.ExecuteTemplate(w, "index.html.tmpl", data)

	log.Printf("File uploaded successfully: %s (%.2f MB)", header.Filename, float64(header.Size)/(1024*1024))
}

// FileInfo represents information about an uploaded file
type FileInfo struct {
	Name     string
	Size     int64
	SizeMB   string
	Modified time.Time
	IsAudio  bool
}

// handleFilesList returns a list of files in the uploads directory
func handleFilesList(w http.ResponseWriter, r *http.Request) {
	uploadsDir := "uploads"

	// Read directory contents
	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, return empty list
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		http.Error(w, "Unable to read uploads directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			ext := strings.ToLower(filepath.Ext(entry.Name()))
			isAudio := ext == ".mp3" || ext == ".wav" || ext == ".m4a"

			files = append(files, FileInfo{
				Name:     entry.Name(),
				Size:     info.Size(),
				SizeMB:   fmt.Sprintf("%.2f", float64(info.Size())/(1024*1024)),
				Modified: info.ModTime(),
				IsAudio:  isAudio,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	// Simple JSON response
	w.Write([]byte("["))
	for i, file := range files {
		if i > 0 {
			w.Write([]byte(","))
		}
		jsonStr := fmt.Sprintf(`{"name":"%s","size":%d,"sizeMB":"%s","modified":"%s","isAudio":%t}`,
			file.Name, file.Size, file.SizeMB, file.Modified.Format("2006-01-02 15:04:05"), file.IsAudio)
		w.Write([]byte(jsonStr))
	}
	w.Write([]byte("]"))
}

// handleFileServe serves files from the uploads directory
func handleFileServe(w http.ResponseWriter, r *http.Request) {
	// Extract filename from URL path
	filename := strings.TrimPrefix(r.URL.Path, "/files/")
	if filename == "" {
		http.Error(w, "Filename not specified", http.StatusBadRequest)
		return
	}

	// Security: prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("uploads", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set appropriate headers based on file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp3":
		w.Header().Set("Content-Type", "audio/mpeg")
	case ".wav":
		w.Header().Set("Content-Type", "audio/wav")
	case ".m4a":
		w.Header().Set("Content-Type", "audio/mp4")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// handleFileDelete deletes a file from the uploads directory
func handleFileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract filename from URL path
	filename := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if filename == "" {
		http.Error(w, "Filename not specified", http.StatusBadRequest)
		return
	}

	// Security: prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("uploads", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Delete the file
	err := os.Remove(filePath)
	if err != nil {
		log.Printf("Error deleting file %s: %v", filename, err)
		http.Error(w, "Unable to delete file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("File deleted successfully: %s", filename)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true, "message": "File deleted successfully"}`))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"Region": os.Getenv("FLY_REGION"),
		}

		t.ExecuteTemplate(w, "index.html.tmpl", data)
	})
	
	http.HandleFunc("/input", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"Region": os.Getenv("FLY_REGION"),
		}

		t.ExecuteTemplate(w, "index.html.tmpl", data)
	})

	http.HandleFunc("/upload", handleFileUpload)
	http.HandleFunc("/api/files", handleFilesList)
	http.HandleFunc("/api/files/", handleFileDelete)
	http.HandleFunc("/files/", handleFileServe)

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
