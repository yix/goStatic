// This small program is just a small web server created in static mode
// in order to provide the smallest docker image possible

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	// Def of flags
	portPtr                  = flag.Int("port", 8043, "The listening port")
	context                  = flag.String("context", "", "The 'context' path on which files are served, e.g. 'doc' will serve the files at 'http://localhost:<port>/doc/'")
	basePath                 = flag.String("path", "/srv/http", "The path for the static files")
	fallbackPath             = flag.String("fallback", "", "Default fallback file. Either absolute for a specific asset (/index.html), or relative to recursively resolve (index.html)")
	headerFlag               = flag.String("append-header", "", "HTTP response header, specified as `HeaderName:Value` that should be added to all responses.")
	basicAuth                = flag.Bool("enable-basic-auth", false, "Enable basic auth. By default, password are randomly generated. Use --set-basic-auth to set it.")
	setBasicAuth             = flag.String("set-basic-auth", "", "Define the basic auth. Form must be user:password")
	defaultUsernameBasicAuth = flag.String("default-user-basic-auth", "gopher", "Define the user")
	sizeRandom               = flag.Int("password-length", 16, "Size of the randomized password")
	statusEnvConfig          = flag.String("status-vars", "", "<ENV_VAR>:<STATUS_VAR>,...")
	statusPath               = flag.String("status-path", "/status", "Path to serve status JSON. Default is /status")
	statusTimestamp          = flag.Bool("status-start-ts", false, "Add start timestamp to the status")

	username string
	password string
)

func parseHeaderFlag(headerFlag string) (string, string) {
	if len(headerFlag) == 0 {
		return "", ""
	}
	pieces := strings.SplitN(headerFlag, ":", 2)
	if len(pieces) == 1 {
		return pieces[0], ""
	}
	return pieces[0], pieces[1]
}

func main() {

	flag.Parse()

	// init status
	StatusInit()

	// sanity check
	if len(*setBasicAuth) != 0 && !*basicAuth {
		*basicAuth = true
	}

	port := ":" + strconv.FormatInt(int64(*portPtr), 10)

	var fileSystem http.FileSystem = http.Dir(*basePath)

	if *fallbackPath != "" {
		fileSystem = fallback{
			defaultPath: *fallbackPath,
			fs:          fileSystem,
		}
	}

	handler := http.FileServer(fileSystem)

	pathPrefix := "/"
	if len(*context) > 0 {
		pathPrefix = "/" + *context + "/"
		handler = http.StripPrefix(pathPrefix, handler)
	}

	if *basicAuth {
		log.Println("Enabling Basic Auth")
		if len(*setBasicAuth) != 0 {
			parseAuth(*setBasicAuth)
		} else {
			generateRandomAuth()
		}
		handler = authMiddleware(handler)
	}

	// Extra headers.
	if len(*headerFlag) > 0 {
		header, headerValue := parseHeaderFlag(*headerFlag)
		if len(header) > 0 && len(headerValue) > 0 {
			fileServer := handler
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(header, headerValue)
				fileServer.ServeHTTP(w, r)
			})
		} else {
			log.Println("appendHeader misconfigured; ignoring.")
		}
	}

	http.Handle(pathPrefix, LogHTTP(handler))

	if *statusEnvConfig != "" || *statusTimestamp {
		http.Handle(*statusPath, LogHTTP(http.HandlerFunc(StatusHandler)))
	}

	log.Printf("Listening at 0.0.0.0%v %v...", port, pathPrefix)
	log.Fatalln(http.ListenAndServe(port, nil))
}

type statusWriter struct {
	http.ResponseWriter
	status int
	length int
}

type LogEntry struct {
	Timestamp     time.Time
	Host          string
	RemoteAddr    string
	Method        string
	RequestURI    string
	Proto         string
	Status        int
	ContentLen    int
	UserAgent     string
	LivenessProbe string `json:",omitempty"`
	Duration      time.Duration
}

func (w *statusWriter) Log(entry LogEntry) {
	if (*statusEnvConfig != "" || *statusTimestamp) &&
		entry.RequestURI == *statusPath &&
		entry.LivenessProbe != "" {
		return // do not log if this is a liveness probe to the `status` URI
	}
	buf, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("Failed to parse log entry %+v\n", entry)
	}
	fmt.Println(string(buf))
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	w.length += n
	return n, err
}

func LogHTTP(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := statusWriter{ResponseWriter: w}
		handler.ServeHTTP(&sw, r)
		duration := time.Now().Sub(start)
		remoteAddr := r.RemoteAddr
		if r.Header.Get("X-Forwarded-For") != "" {
			remoteAddr = r.Header.Get("X-Forwarded-For")
		}
		sw.Log(LogEntry{
			Timestamp:     time.Now(),
			Host:          r.Host,
			RemoteAddr:    remoteAddr,
			Method:        r.Method,
			RequestURI:    r.RequestURI,
			Proto:         r.Proto,
			Status:        sw.status,
			ContentLen:    sw.length,
			UserAgent:     r.Header.Get("User-Agent"),
			LivenessProbe: r.Header.Get("Liveness-Probe"),
			Duration:      duration,
		})
	}
}
