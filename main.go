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
		DefaultThumbSize int
	}

	Gallery map[string]*struct {
		Root string
		ThumbPath string
		ThumbURL string
		ThumbSize int
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
		if gallery.ThumbSize == 0 {
			gallery.ThumbSize = Config.Global.DefaultThumbSize
		}
	}
}
