package main

import (
	"time"
)

type GalleryData struct {
	CacheUntil time.Time
	Dirs       []string
	Images     []ImageInfo
}
type GalleryCache struct {
	Paths map[string]GalleryData
}

func NewGalleryCache() *GalleryCache {
	return &GalleryCache{make(map[string]GalleryData)}
}

func (gc *GalleryCache) Get(basePath string) ([]string, []ImageInfo, bool) {
	// Check cache
	gd, ok := gc.Paths[basePath]
	if !ok {
		return nil, nil, false
	}

	// Check expiration time
	if gd.CacheUntil.Before(time.Now()) {
		delete(gc.Paths, basePath)
		return nil, nil, false
	} else {
		return gd.Dirs, gd.Images, true
	}
}

func (gc *GalleryCache) Set(basePath string, dirs []string, images []ImageInfo) {
	gc.Paths[basePath] = GalleryData{
		CacheUntil: time.Now().Add(time.Duration(Config.Global.CacheTime) * time.Second),
		Dirs:       dirs,
		Images:     images,
	}
}
