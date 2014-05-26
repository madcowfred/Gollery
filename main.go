package main

import (
	"code.google.com/p/gcfg"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/op/go-logging"
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
	cache = NewGalleryCache()
	log   = logging.MustGetLogger("gollery")
	tn    = NewThumbnailer()
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

	log.Debug("Reading config from %s", cfgFile)
	err := gcfg.ReadFileInto(&Config, cfgFile)
	if err != nil {
		log.Fatal(err)
	}

	// Update defaults
	for name, gallery := range Config.Gallery {
		gallery.Name = name

		// Update defaults
		if gallery.BaseURL == "" {
			gallery.BaseURL = "/"
		}
		if gallery.ThumbHeight == 0 {
			gallery.ThumbHeight = Config.Global.DefaultThumbHeight
		}
		if gallery.ThumbWidth == 0 {
			gallery.ThumbWidth = Config.Global.DefaultThumbWidth
		}

		gallery.InitThumbDirs()
	}

	// Set up HTTP handling
	r := mux.NewRouter()

	// Serve static files
	r.PathPrefix("/.static/").Handler(http.StripPrefix("/.static", noDirFileServer(http.FileServer(http.Dir("static/")))))
	// Serve image files
	r.PathPrefix("/.images/").Handler(http.StripPrefix("/.images", http.HandlerFunc(ImageHandler)))
	// Serve thumbnail files
	r.PathPrefix("/.thumbs/").Handler(http.StripPrefix("/.thumbs", http.HandlerFunc(ThumbHandler)))
	// Serve galleries
	r.PathPrefix("/").Handler(LogHandler(os.Stdout, http.HandlerFunc(GalleryHandler)))

	http.Handle("/", r)

	log.Info("Listening on %s", Config.Global.Listen)
	if err = http.ListenAndServe(Config.Global.Listen, r); err != nil {
		panic(err)
	}
}

func (g *GalleryConfig) InitThumbDirs() {
	for _, d := range PREFIXES {
		dirPath := path.Join(g.ThumbPath, string(d))
		if err := os.Mkdir(dirPath, 0755); err != nil {
			log.Warning("Mkdir error: %s", err)
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
