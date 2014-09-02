package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dmichellis/gocassos"
	. "github.com/dmichellis/gocassos/logging"
)

func init() {
	http.HandleFunc("/", VerbRouter)
}

func VerbRouter(w_ http.ResponseWriter, r *http.Request) {
	w := NewStatusRecorder(w_)
	defer func() { AccessLog(w, r) }()

	global := cfg.global // protect against config reload
	global.Enter()
	defer global.Leave()

	var op_err error

	switch r.Method {
	case "GET":
		op_err = Get(w, r)
	case "PUT":
		op_err = Put(w, r)
	case "DELETE":
		op_err = Delete(w, r)

	case "HEAD":
		op_err = Head(w, r)

	case "OPTIONS":
		op_err = HealthCheck(w, r)

	default:
		WTF.Printf("Unknown HTTP verb: %s", r.Method)
		http.Error(w, "Verb not implemented", http.StatusMethodNotAllowed)
		return
	}

	// In case of streaming mode, the header will be sent beforehand with
	// an HTTP 200 OK, so catch it at this point and display the appropriate
	// entry on access.log
	if op_err != nil && w.http_status >= 200 && w.http_status <= 299 {
		switch {
		case strings.HasSuffix(op_err.Error(), ": broken pipe"):
			w.http_status = 499 // http://en.wikipedia.org/wiki/List_of_HTTP_status_codes#4xx_Client_Error -> 499 Client Closed Request (Nginx)
		default:
			w.http_status = http.StatusInternalServerError
		}
	}
}

func PutObjMetadataOnHeaders(w http.ResponseWriter, obj *gocassos.Object) {
	for k, v := range obj.Metadata {
		// Yes, I am aware the X-SOMETHING isn't necessary; however, those are
		// just informational fields and free-form on the storage backend, so
		// be extra-careful presenting them.
		w.Header().Add(fmt.Sprintf("X-Bosos-%s", k), v)
	}
}
func Get(w http.ResponseWriter, r *http.Request) error {
	clowncar := cfg.fetchers // protect against config reload
	clowncar.Enter()
	defer clowncar.Leave()

	opts, header_err := ParseBososHeaders(r)

	if header_err != nil {
		http.Error(w, header_err.Error(), http.StatusBadRequest)
		return header_err
	}

	obj, lookup_err := cfg.backend.Lookup(r.RemoteAddr, r.URL.Path)
	if lookup_err != nil {
		http.NotFound(w, r)
		return lookup_err
	}

	defer func() {
		FYI.Printf("[%s] GET: %s status:%s lookup:%0.4fs fetch:%0.4fs", obj.ClientId, obj.FullName(), obj.Status(), obj.LookupTime.Seconds(), obj.FetchTime.Seconds())
	}()

	PutObjMetadataOnHeaders(w, obj)
	w.Header().Add("Bosos-Transfer-Mode", gocassos.TransferModeCodes[opts.TransferMode])
	if !obj.Expiration.IsZero() {
		w.Header().Add("Expires", obj.Expiration.Format(time.RFC1123))
	}
	obj.NewHttpOutputHandler(w, r, opts.TransferMode)
	out_err := obj.Fetch()
	return out_err
}

func Put(w http.ResponseWriter, r *http.Request) error {
	clowncar := cfg.pushers // protect against config reload
	clowncar.Enter()
	defer clowncar.Leave()

	opts, header_err := ParseBososHeaders(r)
	if header_err != nil {
		http.Error(w, header_err.Error(), http.StatusBadRequest)
		return header_err
	}

	if opts.DoNotUpdate {
		if obj, _ := cfg.backend.Lookup("client_requested_update_protection", r.URL.Path); obj != nil {
			http.Error(w, "Object already exists", http.StatusConflict)
		}
	}

	obj, err := cfg.backend.PreparePush(r.RemoteAddr, r.URL.Path)
	if err != nil {
		WTF.Printf("[%s] PUT: Update to %s failed: %s", r.RemoteAddr, r.URL.Path, err)
		if err == gocassos.ErrRefused {
			http.Error(w, err.Error(), http.StatusForbidden)
			return err
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	defer func() {
		FYI.Printf("[%s] PUT: %s status:%s lookup:%0.4fs push:%0.4fs", obj.ClientId, obj.FullName(), obj.Status(), obj.LookupTime.Seconds(), obj.PushTime.Seconds())
	}()

	obj.Metadata["publisher"] = strings.Split(r.RemoteAddr, ":")[0]
	obj.Metadata["uploaded_to"], _ = os.Hostname()

	obj.Expiration = opts.Expiration

	obj.ChunkSize = opts.ChunkSize
	obj.InputHandler = r.Body
	if push_err := obj.Push(); push_err != nil {
		http.Error(w, push_err.Error(), http.StatusInternalServerError)
		return push_err
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "OK - %s %d chunks stored\n", obj.FullName(), obj.NumChunks)
	return nil
}

func Delete(w http.ResponseWriter, r *http.Request) error {
	obj, lookup_err := cfg.backend.Lookup(r.RemoteAddr, r.URL.Path)
	if lookup_err != nil {
		http.NotFound(w, r)
		return lookup_err
	}
	if del_err := obj.Remove(); del_err != nil {
		if del_err == gocassos.ErrRefused {
			http.Error(w, del_err.Error(), http.StatusForbidden)
			return del_err
		}
		http.Error(w, del_err.Error(), http.StatusInternalServerError)
		return del_err
	}
	fmt.Fprintf(w, "Object %s deleted", obj.FullName())
	return nil
}

func Head(w http.ResponseWriter, r *http.Request) error {
	obj, lookup_err := cfg.backend.Lookup(r.RemoteAddr, r.URL.Path)
	if lookup_err != nil {
		http.NotFound(w, r)
		return errors.New("Not found")
	}
	defer func() {
		FYI.Printf("[%s] HEAD: %s status:%s lookup:%0.4fs", obj.ClientId, obj.FullName(), obj.Status(), obj.LookupTime.Seconds())
	}()

	PutObjMetadataOnHeaders(w, obj)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Add("ETag", fmt.Sprintf("%s", obj.Nodetag))
	w.Header().Add("Last-Modified", time.Unix(obj.Updated, 0).Format(time.RFC1123))
	w.Header().Add("Content-Length", fmt.Sprintf("%d", obj.ObjectSize))
	return nil
}

func HealthCheck(w http.ResponseWriter, r *http.Request) error {
	if cfg.Lb_file != "" {
		if _, err := os.Stat(cfg.Lb_file); err == nil {
			http.Error(w, "I am still alive but you shouldn't use me anymore", http.StatusNotAcceptable)
			WTF.Printf("[%s] Live check: found Disabled_notification_file [%s]", r.RemoteAddr, cfg.Lb_file)
			return errors.New("lb_file present")
		}
	}
	io.WriteString(w, "I am alive and well\n")
	BTW.Printf("[%s] Live check: Ping? Pong!", r.RemoteAddr)
	return nil
}
