package main

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"
)

// Simple struct for page data
type PageData struct {
	BaseURL string
	Dirs    []string    `json:"dirs"`
	Images  []ImageInfo `json:"images"`
}

func galleryStaticHandler(w http.ResponseWriter, r *http.Request, basePath string) {
	// Check path
	cleanPath := path.Clean(path.Join(basePath, r.URL.Path))
	if !strings.HasPrefix(cleanPath, basePath) {
		http.NotFound(w, r)
		return
	}

	// Serve it
	noDirFileServer(http.FileServer(http.Dir(basePath))).ServeHTTP(w, r)
}

// Serve static images for galleries
func ImageHandler(w http.ResponseWriter, r *http.Request) {
	// Check the gallery header
	g := getGallery(r)
	if g == "" {
		http.NotFound(w, r)
		return
	}
	gallery := Config.Gallery[g]

	galleryStaticHandler(w, r, gallery.ImagePath)
}

// Service static thumbnails for galleries
func ThumbHandler(w http.ResponseWriter, r *http.Request) {
	// Check the gallery header
	g := getGallery(r)
	if g == "" {
		http.NotFound(w, r)
		return
	}
	gallery := Config.Gallery[g]

	galleryStaticHandler(w, r, gallery.ThumbPath)
}

func GalleryHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("GalleryHandler: %s", r.URL.Path)

	// Check the gallery header
	g := getGallery(r)
	if g == "" {
		http.NotFound(w, r)
		return
	}
	gallery := Config.Gallery[g]

	// Check path
	cleanPath := path.Clean(path.Join(gallery.ImagePath, r.URL.Path))
	if !strings.HasPrefix(cleanPath, gallery.ImagePath) {
		http.NotFound(w, r)
		return
	}

	// Scan the directory
	dirs, images, err := tn.ScanFolder(gallery, cleanPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	pd := PageData{gallery.BaseURL, dirs, images}
	jsonData, err := json.Marshal(pd)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	log.Debug("%s", jsonData)
}
