package main

import (
	"html/template"
	"net/http"
	"path"
	"strings"
)

var tmpl = make(map[string]*template.Template)

func init() {
	tmpl["gallery"] = template.Must(template.ParseFiles("assets/templates/gallery.html", "assets/templates/base.html"))
}

type Page struct {
	BaseURL string
	JSON    string
	Path    string
	Dirs    []string
	Images  []ImageInfo
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the page
	p := &Page{
		BaseURL: gallery.BaseURL,
		Path:    r.URL.Path,
		Dirs:    dirs,
		Images:  images,
	}
	renderTemplate(w, "gallery", p)
}

// Render a template
func renderTemplate(w http.ResponseWriter, t string, p *Page) {
	err := tmpl[t].ExecuteTemplate(w, "base", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
