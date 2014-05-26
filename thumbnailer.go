package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	reDimensions = regexp.MustCompile(" ([0-9]+)x([0-9]+)")
	reImage      = regexp.MustCompile("(?i)^(.+)\\.(gif|jpeg|jpg|png)$")
)

// Image information, gasp
type ImageInfo struct {
	FileSize    int64  `json:"s"`
	ModTime     int64  `json:"m"`
	ImagePath   string `json:"i"`
	ImageWidth  int    `json:"w"`
	ImageHeight int    `json:"h"`
	ThumbPath   string `json:"t"`
}

type Thumbnailer struct {
	*sync.Mutex
	Paths map[string]*sync.Mutex
}

func NewThumbnailer() *Thumbnailer {
	return &Thumbnailer{
		&sync.Mutex{},
		make(map[string]*sync.Mutex),
	}
}

// Get or create a mutex for a path
func (t *Thumbnailer) GetMutex(basePath string) *sync.Mutex {
	t.Lock()
	defer t.Unlock()

	m, ok := t.Paths[basePath]
	if !ok {
		m := &sync.Mutex{}
		t.Paths[basePath] = m
		return m
	} else {
		return m
	}
}

func (t *Thumbnailer) ScanFolder(gallery *GalleryConfig, basePath string) ([]string, []ImageInfo, error) {
	// start := time.Now()
	// defer func() {
	// 	log.Info("ScanFolder(%s) took %s", basePath, time.Since(start))
	// }()

	// Acquire lock
	m := t.GetMutex(basePath)
	m.Lock()
	defer m.Unlock()

	// Check cache
	cacheDirs, cacheImages, cacheOk := cache.Get(basePath)
	if cacheOk {
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
	fileMap, err := getFileMap(conn, basePath)
	if err != nil {
		return nil, nil, err
	}

	// Some things
	resizeStr := fmt.Sprintf("%dx%d^", gallery.ThumbWidth, gallery.ThumbHeight)
	extentStr := fmt.Sprintf("%dx%d", gallery.ThumbWidth, gallery.ThumbHeight)

	// Iterateee
	// t3 := time.Now()
	for _, fileInfo := range fileNames {
		tl := time.Now()

		fileName := fileInfo.Name()

		// Directories don't need any further processing
		if fileInfo.IsDir() {
			// Skip dotdirectories
			if !strings.HasPrefix(fileName, ".") {
				dirs = append(dirs, fileName)
			}
			continue
		}

		// Don't care about weird filetypes
		if !reImage.MatchString(fileName) {
			continue
		}

		// Check to see if the image has changed
		fileModTime := fileInfo.ModTime().Unix()
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
		// t := time.Now()

		cmd := exec.Command("convert", fmt.Sprintf("%s[0]", filePath), "-thumbnail", resizeStr, "-gravity", "center", "-quality", "90", "-extent", extentStr, "-verbose", thumbPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, nil, err
		}

		// Get image dimensions from the output
		matches := reDimensions.FindAllStringSubmatch(string(out), -1)
		if len(matches) == 0 {
			log.Warning("matches failed: %q", out)
			return nil, nil, err
		}

		imageWidth, err := strconv.ParseInt(matches[0][1], 10, 32)
		if err != nil {
			return nil, nil, err
		}
		imageHeight, err := strconv.ParseInt(matches[0][2], 10, 32)
		if err != nil {
			return nil, nil, err
		}

		// log.Debug("thumbnail for %s took %s", filePath, time.Since(t))

		// Finish junk
		imagePart, _ := filepath.Rel(gallery.ImagePath, filePath)

		imageInfo = ImageInfo{
			FileSize:    fileSize,
			ModTime:     fileModTime,
			ImagePath:   imagePart,
			ImageWidth:  int(imageWidth),
			ImageHeight: int(imageHeight),
			ThumbPath:   thumbPart,
		}
		images = append(images, imageInfo)
		fileMap[fileName] = imageInfo

		log.Debug("loop for %s took %s", filePath, time.Since(tl))
	}
	// log.Debug("Loop took %s", time.Since(t3))

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

func getFileMap(conn redis.Conn, basePath string) (map[string]ImageInfo, error) {
	fileMap := make(map[string]ImageInfo)

	jsonData, err := redis.String(conn.Do("HGET", "images", basePath))
	if err == redis.ErrNil {
		return fileMap, nil
	} else if err != nil {
		return nil, err
	}

	// Try unmarshalling
	if jsonData != "" {
		if err = json.Unmarshal([]byte(jsonData), &fileMap); err != nil {
			return nil, err
		}
	}

	return fileMap, err
}
