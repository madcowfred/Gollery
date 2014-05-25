package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/quirkey/magick"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var reImage = regexp.MustCompile("(?i)^(.+)\\.(gif|jpeg|jpg|png)$")

type ImageInfo struct {
	FileSize    int64
	ModTime     int64
	ImagePath   string
	ImageWidth  int
	ImageHeight int
	ThumbPath   string
}

type Thumbnailer struct {
	*sync.Mutex
}

func NewThumbnailer() *Thumbnailer {
	return &Thumbnailer{&sync.Mutex{}}
}

func (t *Thumbnailer) ScanFolder(gallery *GalleryConfig, basePath string) ([]string, []ImageInfo, error) {
	start := time.Now()
	defer func() {
		log.Info("ScanFolder(%s) took %s", basePath, time.Since(start))
	}()

	// Acquire lock
	t.Lock()
	defer t.Unlock()

	// Check cache
	cacheDirs, cacheImages, cacheOk := cache.Get(basePath)
	if cacheOk {
		log.Debug("cached")
		return cacheDirs, cacheImages, nil
	}

	// Get a Redis connection
	conn := redisPool.Get()
	defer conn.Close()

	// Vars
	var dirs []string
	var images []ImageInfo

	// Get the files
	fileNames, err := ioutil.ReadDir(basePath)
	if err != nil {
		return nil, nil, err
	}

	// Subfolders need a fake .. directory
	if basePath != gallery.ImagePath {
		dirs = append(dirs, "..")
	}

	// Try fetching data from Redis
	t1 := time.Now()
	jsonData, err := redis.String(conn.Do("HGET", "images", basePath))
	if err != redis.ErrNil && err != nil {
		return nil, nil, err
	}
	log.Debug("HGET took %s", time.Since(t1))

	// Try unmarshalling
	t2 := time.Now()
	fileMap := make(map[string]ImageInfo)
	if jsonData != "" {
		if err = json.Unmarshal([]byte(jsonData), &fileMap); err != nil {
			return nil, nil, err
		}
	}
	log.Debug("Unmarshal took %s", time.Since(t2))

	// Some things
	resizeStr := fmt.Sprintf("%dx%d^", gallery.ThumbWidth, gallery.ThumbHeight)
	extentStr := fmt.Sprintf("%dx%d", gallery.ThumbWidth, gallery.ThumbHeight)

	// Iterateee
	t3 := time.Now()
	for _, fileInfo := range fileNames {
		fileName := fileInfo.Name()

		// Don't care about directories
		if fileInfo.IsDir() {
			dirs = append(dirs, fileName)
			continue
		}

		// Check to see if the image has changed
		fileModTime := fileInfo.ModTime().UnixNano()
		fileSize := fileInfo.Size()

		imageInfo, ok := fileMap[fileName]
		if ok && imageInfo.FileSize == fileSize && imageInfo.ModTime == fileModTime && imageInfo.ThumbPath != "" {
			images = append(images, imageInfo)
			continue
		}

		filePath := path.Join(basePath, fileName)

		// Generate the thumbnail filename and path
		b, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, nil, err
		}
		thumbName := fmt.Sprintf("%x.jpg", md5.Sum(b))
		thumbPart := path.Join(string(thumbName[0]), thumbName)
		thumbPath := path.Join(gallery.ThumbPath, thumbPart)

		// Generate the thumbnail image and save it
		t := time.Now()

		cmd := exec.Command("convert", filePath, "-thumbnail", resizeStr, "-gravity", "center", "-extent", extentStr, thumbPath)
		if err = cmd.Run(); err != nil {
			return nil, nil, err
		}

		// Read image dimensions :(
		img1, err := magick.NewFromFile(filePath)
		defer img1.Destroy()
		if err != nil {
			return nil, nil, err
		}

		imageWidth, imageHeight := img1.Width(), img1.Height()

		log.Debug("thumbnail for %s took %s", filePath, time.Since(t))

		// Finish junk
		imagePart, _ := filepath.Rel(gallery.ImagePath, filePath)

		imageInfo = ImageInfo{
			FileSize:    fileSize,
			ModTime:     fileModTime,
			ImagePath:   imagePart,
			ImageWidth:  imageWidth,
			ImageHeight: imageHeight,
			ThumbPath:   thumbPart,
		}
		images = append(images, imageInfo)
		fileMap[fileName] = imageInfo
	}
	log.Debug("Loop took %s", time.Since(t3))

	// Update cache
	cache.Set(basePath, dirs, images)

	// Update Redis
	b, err := json.Marshal(fileMap)
	if err != nil {
		return nil, nil, err
	}
	conn.Do("HSET", "images", basePath, string(b))

	return dirs, images, nil
}
