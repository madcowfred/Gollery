package main

import (
	"net/http"
	"path"
	"strings"
)

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

	log.Debug("%s | %s", gallery.ImagePath, cleanPath)
}
