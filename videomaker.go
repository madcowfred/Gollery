package main

import (
	"bytes"
	"fmt"
	"image/gif"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"
)

//ffmpeg -i kitty_bloopers.gif -c:v libvpx -threads 0 -an -crf 10 ~/public_html/test.webm

var (
	reGIF = regexp.MustCompile("(?i)^(.+)\\.(gif)$")
	test  = "\x00\x21\xF9\x04baaa\x00\x2Czzzzzzzz\x00\x21\xF9\x04boooo\x00\x2C"
)

func VideoMaker() chan FolderData {
	// Get a Redis connection
	conn := redisPool.Get()
	defer conn.Close()

	c := make(chan FolderData, 1000)

	go func() {
		var start time.Time

		for fd := range c {
			start = time.Now()

			key := fmt.Sprintf("webm:%s", fd.BasePath)

			// Bail if this gallery doesn't have a video path
			if fd.Gallery.VideoPath == "" {
				log.Debug("VideoMaker(%s) has no VideoPath configured", fd.BasePath)
				continue
			}

			for fileName, imageInfo := range *fd.FileMap {
				t := time.Now()

				// Don't care about non-GIFs
				fileMatches := reGIF.FindAllStringSubmatch(fileName, -1)
				if len(fileMatches) == 0 {
					continue
				}

				// See if the video file already exists
				videoName := strings.Replace(imageInfo.ThumbPath, ".jpg", ".webm", 1)
				videoPath := path.Join(fd.Gallery.VideoPath, videoName)
				if _, err := os.Stat(videoPath); err == nil {
					//log.Debug("VideoMaker(%s) file exists %s", fd.BasePath, videoPath)
					continue
				}

				//log.Debug("VideoMaker(%s) file does not exist %s", fd.BasePath, videoPath)

				// asdf
				filePath := path.Join(fd.BasePath, fileName)

				// Read the file
				b, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Warning("VideoMaker(%s) unable to read file %s: %s", fd.BasePath, fileName, err.Error())
					continue
				}

				// Decode the GIF
				buf := bytes.NewBuffer(b)
				g, err := gif.DecodeAll(buf)
				if err != nil {
					log.Warning("VideoMaker(%s) unable to decode GIF %s: %s", fd.BasePath, fileName, err.Error())
					continue
				}

				// Skip non-animated GIFs
				if len(g.Image) <= 1 {
					log.Debug("VideoMaker(%s) not animated %s", fd.BasePath, fileName)
					continue
				}

				// Now we can finally make a webm
				cmd := exec.Command("ffmpeg", "-i", filePath, "-c:v", "libvpx", "-threads", "0", "-an", "-crf", "10", videoPath)
				if err = cmd.Run(); err != nil {
					log.Warning("VideoMaker(%s) unable to make webm %s: %s", fd.BasePath, fileName, err.Error())
					continue
				}

				// Save to Redis
				conn.Do("HSET", key, imageInfo.ImagePath, videoName)

				log.Debug("VideoMaker(%s) webm of %s took %s", fd.BasePath, fileName, time.Since(t))
			}

			log.Debug("VideoMaker(%s) took %s", fd.BasePath, time.Since(start))
		}
	}()

	return c
}
