package main

import (
	"code.google.com/p/gcfg"
	"github.com/op/go-logging"
	"path/filepath"
	// "time"
)

var log = logging.MustGetLogger("gollery")

var Config struct {
	Global struct {
		DefaultThumbWidth int
		DefaultThumbHeight int
	}

	Gallery map[string]*struct {
		ImagePath string
		ImageURL string
		ThumbPath string
		ThumbURL string
		ThumbWidth int
		ThumbHeight int
	}
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
	for _, gallery := range Config.Gallery {
		if gallery.ThumbHeight == 0 {
			gallery.ThumbHeight = Config.Global.DefaultThumbHeight
		}
		if gallery.ThumbWidth == 0 {
			gallery.ThumbWidth = Config.Global.DefaultThumbWidth
		}
	}
}
