package main

import (
	"code.google.com/p/gcfg"
	"crypto/md5"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/op/go-logging"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	PREFIXES = "0123456789abcdef"
)

var (
	cache       = NewGalleryCache()
	log         = logging.MustGetLogger("gollery")
	staticFiles = make(map[string]string)
	tn          = NewThumbnailer()
)

// Redis connection pool
var redisPool = &redis.Pool{
	MaxIdle:     2,
	IdleTimeout: 60 * time.Second,
	Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", Config.Redis.ConnectionString)
		if err != nil {
			return nil, err
		}
		c.Do("SELECT", Config.Redis.Database)
		return c, err
	},
	TestOnBorrow: func(c redis.Conn, t time.Time) error {
		_, err := c.Do("PING")
		return err
	},
}

// Config stuff
type GalleryConfig struct {
	Name        string
	BaseURL     string
	ImagePath   string
	ThumbPath   string
	ThumbWidth  int
	ThumbHeight int
}

var Config struct {
	Global struct {
		Listen             string
		CacheTime          int
		DefaultThumbWidth  int
		DefaultThumbHeight int
	}

	Redis struct {
		ConnectionString string
		Database         int
	}

	Gallery map[string]*GalleryConfig
}

func main() {
	// Set up logging
	var format = logging.MustStringFormatter(" %{level: -8s}  %{message}")
	logging.SetFormatter(format)
	logging.SetLevel(logging.DEBUG, "gollery")
	// logging.SetLevel(logging.INFO, "gmc")

	log.Info("Gollery starting...")

	// Load config file
	var cfgFile = filepath.Join(".", "gollery.conf")

	log.Info("Reading config from %s", cfgFile)
	err := gcfg.ReadFileInto(&Config, cfgFile)
	if err != nil {
		log.Fatal(err)
	}

	// Update defaults
	for name, gallery := range Config.Gallery {
		// Update defaults
		if gallery.BaseURL == "" {
			gallery.BaseURL = "/"
		}
		if gallery.Name == "" {
			gallery.Name = name
		}
		if gallery.ThumbHeight == 0 {
			gallery.ThumbHeight = Config.Global.DefaultThumbHeight
		}
		if gallery.ThumbWidth == 0 {
			gallery.ThumbWidth = Config.Global.DefaultThumbWidth
		}

		gallery.InitThumbDirs()
	}

	// Start the cache expire timer
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for _ = range ticker.C {
			cache.Expire()
		}
	}()

	// Generate hashed filenames for static files
	fileNames, err := ioutil.ReadDir("static")
	if err != nil {
		log.Fatal(err)
	}

	for _, fileInfo := range fileNames {
		fileName := fileInfo.Name()
		filePath := path.Join("static", fileName)

		b, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}

		ext := path.Ext(fileName)
		staticFiles[fileName] = strings.Replace(fileName, ext, fmt.Sprintf(".%x%s", md5.Sum(b), ext), 1)
	}

	// Set up HTTP handling
	r := mux.NewRouter()

	// Serve static files
	r.HandleFunc("/favicon.ico", serveStatic)
	r.HandleFunc("/robots.txt", serveStatic)

	for fileName, hashName := range staticFiles {
		r.Path("/.static/" + hashName).Handler(expiresHandler(30, staticHandler(fileName)))
	}
	//r.PathPrefix("/.static/").Handler(http.StripPrefix("/.static", noDirFileServer(http.FileServer(http.Dir("static/")))))

	// Serve image files
	r.PathPrefix("/.images/").Handler(http.StripPrefix("/.images", http.HandlerFunc(ImageHandler)))
	// Serve thumbnail files
	r.PathPrefix("/.thumbs/").Handler(http.StripPrefix("/.thumbs", expiresHandler(30, http.HandlerFunc(ThumbHandler))))
	// Serve galleries
	r.PathPrefix("/").Handler(LogHandler(os.Stdout, http.HandlerFunc(GalleryHandler)))

	http.Handle("/", r)

	// Listen and serve
	log.Info("Listening on %s", Config.Global.Listen)
	if err = http.ListenAndServe(Config.Global.Listen, r); err != nil {
		panic(err)
	}
}

func (g *GalleryConfig) InitThumbDirs() {
	for _, d := range PREFIXES {
		dirPath := path.Join(g.ThumbPath, string(d))
		if err := os.Mkdir(dirPath, 0755); err != nil {
			if !os.IsExist(err) {
				log.Warning("Mkdir error: %s", err)
			}
		}
	}
}

func noDirFileServer(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Serve a static file using the URL path
func serveStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join("static", r.URL.Path))
}

// Serve a static file using a specific filename
func staticHandler(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("static", filename))
	})
}

// Dumb expires handler
func expiresHandler(days int, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cache-Control
		secs := days * 24 * 60 * 60
		w.Header().Add("Cache-Control", fmt.Sprintf("max-age=%d", secs))

		// Expires
		expires := time.Now().AddDate(0, 0, 30).UTC().Format(time.RFC1123)
		w.Header().Add("Expires", expires)

		h.ServeHTTP(w, r)
	})
}

func getGallery(r *http.Request) string {
	var gallery string

	head, ok := r.Header["X-Gollery"]
	if ok {
		if _, ok := Config.Gallery[head[0]]; ok {
			gallery = head[0]
		}
	}

	return gallery
}
