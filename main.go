package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
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
			img, _, err := image.Decode(file)
			if err != nil {
				e = true
				fmt.Fprintf(w, "2. %v", err)
			} else {
				writeImage(w, &img)
			}
		} else {
			e = true
		}
		if e == true {
			w.WriteHeader(404)
		}
	}

}

func writeImage(w http.ResponseWriter, img *image.Image) {

	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, *img); err != nil {
		log.Println("unable to encode image.")
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Println("unable to write image.")
	}
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	file, handle, err := r.FormFile("data")
	if err != nil {
		fmt.Fprintf(w, "0. %v", err)
		return
	}
	defer file.Close()
	saveFile(w, file, handle)
}

func saveFile(w http.ResponseWriter, file multipart.File, handle *multipart.FileHeader) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Fprintf(w, "3. %v", err)
		return
	}

	name := RandStringRunes(16)
	for {
		if _, err := os.Stat("./files/" + name); !os.IsNotExist(err) {
			name = RandStringRunes(16)
		} else {
			break
		}
	}

	err = ioutil.WriteFile("./files/"+name, data, 0666)
	if err != nil {
		fmt.Fprintf(w, "4. %v", err)
		return
	}
	jsonResponse(w, http.StatusCreated, Response{
		URI: "http://" + os.Getenv("VIRTUAL_HOST") + "/" + name,
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
	URI    string `json:"uri,omitempty"`
	DelURI string `json:"del_uri,omitempty"`
}
