package main

import (
	"sync"
)

type Image struct {
	Filename string
	ImageURL string
	ThumbURL string
	Width int
	Height int
}

type Thumbnailer struct {
	*sync.Mutex
}

func (t *Thumbnailer) ScanFolder(filepath string) ([]Image, error) {
	// Acquire lock
	t.Lock()
	defer t.Unlock()

	return nil, nil
}
