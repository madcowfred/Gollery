package main

import (
	"io"
	"net"
	"net/http"
	"time"
)

type logHandler struct {
	writer  io.Writer
	handler http.Handler
}

// This is basically a copy/paste of the gorilla/handlers code with a different log format
func (h logHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Info("loghandler")

	t := time.Now()
	logger := &responseLogger{w: w}
	h.handler.ServeHTTP(logger, req)

	// Write the log message
	var host string

	if realIP, ok := req.Header["X-Real-Ip"]; ok {
		host = realIP[0]
	} else {
		t, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			host = req.RemoteAddr
		} else {
			host = t
		}
	}

	// Log it
	log.Info("\"%s\" %s \"%s %s %s\" %d %d -- %s",
		getGallery(req),
		host,
		req.Method,
		req.URL.RequestURI(),
		req.Proto,
		logger.Status(),
		logger.Size(),
		time.Since(t),
	)
	//writeLog(h.writer, req, t, logger.Status(), logger.Size())
}

func LogHandler(out io.Writer, h http.Handler) http.Handler {
	log.Info("LogHandler()")
	return logHandler{out, h}
}


// responseLogger is wrapper of http.ResponseWriter that keeps track of its HTTP status
// code and body size
type responseLogger struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (l *responseLogger) Header() http.Header {
	return l.w.Header()
}

func (l *responseLogger) Write(b []byte) (int, error) {
	if l.status == 0 {
		// The status will be StatusOK if WriteHeader has not been called yet
		l.status = http.StatusOK
	}
	size, err := l.w.Write(b)
	l.size += size
	return size, err
}

func (l *responseLogger) WriteHeader(s int) {
	l.w.WriteHeader(s)
	l.status = s
}

func (l *responseLogger) Status() int {
	return l.status
}

func (l *responseLogger) Size() int {
	return l.size
}
