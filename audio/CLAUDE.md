# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go web application called "waveform" that provides MP3 audio file upload and waveform generation capabilities. The application accepts MP3 uploads, processes them to generate waveform data, and provides a REST API for file management.

## Common Commands

### Development
```bash
# Install/update dependencies
go mod tidy

# Build the application
go build .

# Run the application locally
go run main.go

# Build and run Docker container
docker build -t waveform .
docker run -p 8080:8080 waveform
```

### Deployment
```bash
# Deploy to Fly.io (configured in fly.toml)
fly deploy
```

## Architecture Overview

### Core Components

**main.go** - HTTP server with these key endpoints:
- `/` and `/input` - Serve HTML upload interface
- `/upload` (POST) - Handle MP3 file uploads with 500MB limit
- `/api/files` (GET) - List uploaded files as JSON
- `/api/files/{filename}` (DELETE) - Delete files
- `/files/{filename}` (GET) - Serve/download files

**internal/run.go** - Audio processing engine:
- Uses `github.com/hopesea/godub` library for MP3 processing
- Analyzes audio in 220 frequency bands
- Calculates max/avg values for each band
- Outputs binary waveform data (.bin files)

**templates/index.html.tmpl** - Single-page web interface:
- TailwindCSS for styling
- File upload with progress tracking
- CAPTCHA security validation
- Real-time file management (list/delete)
- AJAX-based interactions

### File Flow
1. User uploads MP3 via web form
2. File saved to `uploads/` directory
3. Background goroutine processes file with `internal.Run()`
4. Original MP3 deleted after processing
5. Binary waveform file (.avg220.bin) generated
6. Files accessible via `/files/` endpoint

### Key Dependencies
- `github.com/hopesea/godub` - Audio processing
- Embedded templates via `//go:embed`
- Standard library HTTP server

### Configuration
- **Port**: Environment variable `PORT` (default: 8080)
- **Max file size**: 500MB hardcoded limit
- **Upload directory**: `uploads/` (created automatically)
- **Fly.io region**: Displayed in UI from `FLY_REGION` env var

### Security Features
- File type validation (MP3 only)
- File size limits (500MB)
- Directory traversal protection
- Math CAPTCHA on uploads
- CSRF-style validation

### Processing Details
The waveform generation splits audio into 220 bands and calculates `(max - avg) / 2` for each band, then outputs as little-endian 16-bit integers with duration appended. Output files follow naming pattern: `{original_filename}.avg220.bin`.