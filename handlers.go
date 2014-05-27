package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

var tmpl = make(map[string]*template.Template)

const (
	TIME_FORMAT = "2006-01-02 15:04:05"
)

func init() {
	tmpl["gallery"] = template.Must(template.New("gallery").
		Funcs(template.FuncMap{
		"formatSize": formatSize,
		"formatTime": formatTime,
	}).
		ParseFiles("assets/templates/gallery.html", "assets/templates/base.html"))
}

type DirInfo struct {
	Path string
	Name string
}

type Page struct {
	BaseURL      string
	JSON         string
	Name         string
	Path         string
	StaticFolder string
	StaticCSS    string
	StaticJS     string
	Dirs         []DirInfo
	Images       []ImageInfo
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

// Serve static thumbnails for galleries
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

// Serve a gallery page
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

	// Check for trailing /
	if !strings.HasSuffix(r.URL.Path, "/") {
		newPath := path.Clean(gallery.BaseURL+r.URL.Path) + "/"
		log.Debug("%s -> %s", r.URL.Path, newPath)
		localRedirect(w, r, newPath)
		return
	}

	// Scan the directory
	dirs, images, err := tn.ScanFolder(gallery, cleanPath)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Error("GalleryHandler:", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Zzrp
	var dirinfos []DirInfo
	for _, dirPath := range dirs {
		dirinfos = append(dirinfos, DirInfo{dirPath, strings.Replace(dirPath, "_", " ", -1)})
	}

	// Render the page
	p := &Page{
		BaseURL:      gallery.BaseURL,
		Name:         gallery.Name,
		Path:         r.URL.Path,
		StaticCSS:    staticFiles["gollery.min.css"],
		StaticFolder: staticFiles["folder.png"],
		StaticJS:     staticFiles["gollery.min.js"],
		Dirs:         dirinfos,
		Images:       images,
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

// (from net/http/fs.go)
//
// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like Redirect does.
func localRedirect(w http.ResponseWriter, r *http.Request, newPath string) {
	if q := r.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	w.Header().Set("Location", newPath)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Simple pipeline func to format file size nicely
func formatSize(size int64) string {
	if size < (1024 * 1024) {
		return fmt.Sprintf("%.1f KiB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MiB", float64(size)/1024/1024)
	}
}

// Simple pipeline func to format unix time nicely
func formatTime(unix int64) string {
	return time.Unix(unix, 0).Format(TIME_FORMAT)
}
