package main

import (
	"net/http"
	"path"
	"strings"
)

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
