package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dmichellis/cronologo"
	//	. "github.com/dmichellis/gocassos/logging"
)

type httpStatusRecorder struct {
	http_status int

	w http.ResponseWriter
}

func NewStatusRecorder(w_ http.ResponseWriter) *httpStatusRecorder {
	return &httpStatusRecorder{http_status: -1, w: w_}
}

func (w *httpStatusRecorder) Header() http.Header { return w.w.Header() }

func (w *httpStatusRecorder) Write(d []byte) (int, error) {
	if w.http_status == -1 {
		w.http_status = http.StatusOK
	}
	return w.w.Write(d)
}

func (w *httpStatusRecorder) WriteHeader(code int) {
	w.http_status = code
	w.w.WriteHeader(code)
}

func AccessLog(w *httpStatusRecorder, r *http.Request) {
	var size int
	switch r.Method {
	case "PUT":
		size = int(r.ContentLength)
	default:
		size, _ = strconv.Atoi(w.Header().Get("Content-Length"))
	}
	access_log.Printf("%s - - [%s] \"%s %s %s\" %d %d \"-\" \"%s\" \"-\"\n", strings.Split(r.RemoteAddr, ":")[0], time.Now().UTC().Format("_2/Jul/2006 15:04:05 -0700"), r.Method, r.URL.Path, r.Proto, w.http_status, size, r.Header.Get("User-Agent"))
}

var access_log *log.Logger

var AccessLogR = cronologo.LogFile{
	NamePrefix: "",
	TimeFormat: "2006-01-02",
	Symlink:    true,
	CallBack: func(f *os.File) {
		access_log = log.New(f, "", 0)
	},
}

func init() {
	access_log = log.New(os.Stdout, "", 0)
}
