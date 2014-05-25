package main

import (
	"code.google.com/p/gcfg"
	"github.com/garyburd/redigo/redis"
	"github.com/op/go-logging"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	PREFIXES = "0123456789abcdef"
)

var (
	cache = NewGalleryCache()
	log = logging.MustGetLogger("gollery")
	tn = NewThumbnailer()
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
	ImagePath   string
	ImageURL    string
	ThumbPath   string
	ThumbURL    string
	ThumbWidth  int
	ThumbHeight int
}
var Config struct {
	Global struct {
		CacheTime          int
		DefaultThumbWidth  int
		DefaultThumbHeight int
	}

	Redis struct {
		ConnectionString string
		Database int
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

		// Update thumb defaults
		if gallery.ThumbHeight == 0 {
			gallery.ThumbHeight = Config.Global.DefaultThumbHeight
		}
		if gallery.ThumbWidth == 0 {
			gallery.ThumbWidth = Config.Global.DefaultThumbWidth
		}

		initThumbDirs(gallery)

		dirs, images, err := tn.ScanFolder(gallery, path.Join(gallery.ImagePath, "pets"))
		if err != nil {
			log.Fatal(err)
		}

		log.Debug("dirs: %v", dirs)
		log.Debug("images: %v", images)
	}

	// tn.ScanFolder(Config.Gallery[0])
}

func initThumbDirs(gallery *GalleryConfig) {
	for _, d := range PREFIXES {
		dirPath := path.Join(gallery.ThumbPath, string(d))
		if err := os.Mkdir(dirPath, 0755); err != nil {
			log.Warning("Mkdir error: %s", err)
		}
	}
}
