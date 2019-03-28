package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type RegexpHandler struct {
	routes []*route
}

func (h *RegexpHandler) Handler(pattern *regexp.Regexp, handler http.Handler) {
	h.routes = append(h.routes, &route{pattern, handler})
}

func (h *RegexpHandler) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
	h.routes = append(h.routes, &route{pattern, http.HandlerFunc(handler)})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) {
			log.WithFields(log.Fields{
				"URI":        r.URL.Path,
				"IP":         RemoteAddr(r),
				"User-Agent": r.Header.Get("User-Agent"),
				"Referer":    r.Header.Get("Referer"),
			}).Info("Request URI")
			route.handler.ServeHTTP(w, r)
			return
		}
	}

	http.NotFound(w, r)
}

func RemoteAddr(r *http.Request) string {

	addr := r.Header.Get("X-Real-IP")
	if len(addr) == 0 {
		addr = r.Header.Get("X-Forwarded-For")
		if addr == "" {
			addr = r.RemoteAddr
			if i := strings.LastIndex(addr, ":"); i > -1 {
				addr = addr[:i]
			}
		}
	}
	return addr
}

var VirtualHost string

func init() {
	rand.Seed(time.Now().UnixNano())
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)

	log.Info("Start Sharex Uploader")
}

func main() {
	VirtualHost = os.Getenv("VIRTUAL_HOST")
	handler := &RegexpHandler{}
	handler.HandleFunc(regexp.MustCompile(`\/upload`), UploadFile)
	handler.HandleFunc(regexp.MustCompile(`\/[a-zA-Z0-9]{16}\.(.*)`), ShowFile)

	log.WithFields(log.Fields{
		"VirtualHost": VirtualHost,
	}).Fatal(http.ListenAndServe(":3000", handler))
}

var letterRunes = []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func UploadFile(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {
		file, handle, err := r.FormFile("data")
		if err != nil {
			log.WithFields(log.Fields{
				"URI":   r.URL.Path,
				"IP":    RemoteAddr(r),
				"Error": err,
			}).Info("Can't get file in the request")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		mimeType := handle.Header.Get("Content-Type")
		switch mimeType {
		case "image/jpeg", "image/png", "image/gif":
			saveFile(w, file, handle)
		default:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	} else {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}
}

func ShowFile(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		filename := strings.Split(r.URL.Path, "/")[1]
		path := filepath.Join("./files", filename)
		e := false
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			if file, err := os.Open(path); err != nil {
				e = true
			} else {
				buff := make([]byte, 512)
				file.Seek(0, 0)
				_, err = file.Read(buff)
				file.Seek(0, 0)
				if err != nil && err != io.EOF {
					e = true
				} else {
					w.Header().Set("Content-Type", http.DetectContentType(buff))
					fi, err := file.Stat()
					if err != nil {
						log.WithFields(log.Fields{
							"File": log.Fields{
								"Filename": file.Name(),
							},
							"Error": err,
						}).Warn("Can't read file size")
						w.WriteHeader(http.StatusInternalServerError)
					} else {
						buff := make([]byte, fi.Size())
						file.Read(buff)
						w.Header().Set("Content-Length", strconv.Itoa(len(buff)))
						if _, err := w.Write(buff); err != nil {
							log.WithFields(log.Fields{
								"Response": log.Fields{
									"Filename": fi.Name(),
									"Size":     fi.Size(),
									"Mod":      fi.Mode().String(),
								},
								"Error": err,
							}).Warn("Unable to write file in response")
							w.WriteHeader(http.StatusInternalServerError)
						}
					}
				}
			}
		} else {
			e = true
		}
		if e == true {
			http.NotFound(w, r)
		}
	} else {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}
}

func saveFile(w http.ResponseWriter, file multipart.File, handle *multipart.FileHeader) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.WithFields(log.Fields{
			"Handle": log.Fields{
				"Filename": handle.Filename,
				"Size":     handle.Size,
				"Headers":  handle.Header,
			},
			"Error": err,
		}).Warn("Can't read temporary file")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	name := RandStringRunes(16)
	fileType := strings.Split(handle.Filename, ".")[1]
	fileName := name + "." + fileType
	for {
		if _, err := os.Stat("./files/" + fileName); !os.IsNotExist(err) {
			name = RandStringRunes(16)
			fileName = name + "." + fileType
		} else {
			break
		}
	}

	err = ioutil.WriteFile("./files/"+fileName, data, 0666)
	if err != nil {
		log.WithFields(log.Fields{
			"File": log.Fields{
				"Filename": fileName,
				"Name":     name,
				"Type":     fileType,
			},
			"Error": err,
		}).Error("Can't write file")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, http.StatusCreated, Response{
		URI: fmt.Sprintf("http://%s/%s", VirtualHost, fileName),
		//DelURI: "http://" + os.Getenv("VirtualHost") + "/delete" + handle.Filename,
	})
}

func jsonResponse(w http.ResponseWriter, code int, message Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(&message)
	if err != nil {
		log.WithFields(log.Fields{
			"Message": message,
			"Code":    code,
			"Error":   err,
		}).Error("Json Response")
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

type Response struct {
	URI          string `json:"uri,omitempty"`
	ThumbnailURI string `json:"thumbnail_uri,omitempty"`
	DelURI       string `json:"del_uri,omitempty"`
}
