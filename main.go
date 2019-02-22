package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())

}
func main() {
	http.HandleFunc("/", upload)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var letterRunes = []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func upload(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {

		UploadFile(w, r)

	}

	if r.Method == http.MethodGet {
		parts := strings.Split(r.URL.Path, "/")
		path := "./files/" + parts[1]
		e := false
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			file, err := os.Open(path)
			if err != nil {
				e = true
				fmt.Fprintf(w, "1. %v", err)
			}

			buff := make([]byte, 512)
			file.Seek(0, 0)
			_, err = file.Read(buff)
			file.Seek(0, 0)

			if err != nil && err != io.EOF {
				e = true
				fmt.Fprintf(w, "5. %v", err)
			} else {
				contentType := http.DetectContentType(buff)
				writeImage(w, file, contentType)
			}
		} else {
			e = true
		}
		if e == true {
			w.WriteHeader(404)
		}
	}

}

func writeImage(w http.ResponseWriter, file *os.File, mime string) {

	w.Header().Set("Content-Type", mime)
	fi, err := file.Stat()
	if err != nil {
		log.Println("cant read file size")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		buff := make([]byte, fi.Size())
		file.Read(buff)
		w.Header().Set("Content-Length", strconv.Itoa(len(buff)))
		if _, err := w.Write(buff); err != nil {
			log.Println("unable to write image.")
		}
	}
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	file, handle, err := r.FormFile("data")
	if err != nil {
		fmt.Fprintf(w, "0. %v", err)
		return
	}
	defer file.Close()

	mimeType := handle.Header.Get("Content-Type")
	switch mimeType {
	case "image/jpeg":
		saveFile(w, file, handle)
	case "image/png":
		saveFile(w, file, handle)
	case "image/gif":
		saveFile(w, file, handle)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}

	//saveFile(w, file, handle)
}

func saveFile(w http.ResponseWriter, file multipart.File, handle *multipart.FileHeader) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Fprintf(w, "3. %v", err)
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
		fmt.Fprintf(w, "4. %v", err)
		return
	}
	jsonResponse(w, http.StatusCreated, Response{
		URI: "http://" + os.Getenv("VIRTUAL_HOST") + "/" + fileName,
		//DelURI: "http://" + os.Getenv("VIRTUAL_HOST") + "/delete" + handle.Filename,
	})
}

func jsonResponse(w http.ResponseWriter, code int, message Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(&message)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
	}
}

type Response struct {
	URI          string `json:"uri,omitempty"`
	ThumbnailURI string `json:"thumbnail_uri,omitempty"`
	DelURI       string `json:"del_uri,omitempty"`
}
