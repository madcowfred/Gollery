package main

import (
	"sync"
	"time"
)

type GalleryData struct {
	CacheUntil time.Time
	Dirs       []string
	Images     []ImageInfo
}
type GalleryCache struct {
	*sync.Mutex
	Paths map[string]GalleryData
}

func NewGalleryCache() *GalleryCache {
	return &GalleryCache{
		&sync.Mutex{},
		make(map[string]GalleryData),
	}
}

func (gc *GalleryCache) Get(basePath string) ([]string, []ImageInfo, bool) {
	// Acquire lock
	gc.Lock()
	defer gc.Unlock()

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
	// Acquire lock
	gc.Lock()
	defer gc.Unlock()

	gc.Paths[basePath] = GalleryData{
		CacheUntil: time.Now().Add(time.Duration(Config.Global.CacheTime) * time.Second),
		Dirs:       dirs,
		Images:     images,
	}
}

func (gc *GalleryCache) Delete(basePath string) {
	delete(gc.Paths, basePath)
}

func (gc *GalleryCache) Expire() {
	// Acquire lock
	gc.Lock()
	defer gc.Unlock()

	now := time.Now()
	for k, gd := range gc.Paths {
		if gd.CacheUntil.Before(now) {
			delete(gc.Paths, k)
		}
	}
}
