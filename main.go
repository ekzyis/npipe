package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

//go:embed public/index.html
var indexHTML string

//go:embed public/style.css
var styleCSS string

//go:embed public/app.js
var appJS string

type File struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Data []byte `json:"-"`
}

type FileStore struct {
	mu    sync.RWMutex
	files map[string]map[string]*File // ip -> id -> file
}

func NewFileStore() *FileStore {
	return &FileStore{files: make(map[string]map[string]*File)}
}

func (s *FileStore) Add(ip, name string, data []byte) string {
	id := make([]byte, 4)
	rand.Read(id)
	fileID := hex.EncodeToString(id)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.files[ip] == nil {
		s.files[ip] = make(map[string]*File)
	}
	s.files[ip][fileID] = &File{ID: fileID, Name: name, Data: data}
	return fileID
}

func (s *FileStore) List(ip string) []*File {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var files []*File
	for _, f := range s.files[ip] {
		files = append(files, f)
	}
	return files
}

func (s *FileStore) Get(ip, id string) *File {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.files[ip] == nil {
		return nil
	}
	return s.files[ip][id]
}

var store = NewFileStore()

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func main() {
	port := "3333"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		html := strings.Replace(indexHTML, "__CLIENT_IP__", ip, 1)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(styleCSS))
	})

	http.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		w.Write([]byte(appJS))
	})

	http.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		files := store.List(ip)
		if files == nil {
			files = []*File{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	})

	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ip := getClientIP(r)
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Name string `json:"name"`
			Data []byte `json:"data"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		store.Add(ip, req.Name, req.Data)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/file/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/file/")
		ip := getClientIP(r)
		file := store.Get(ip, id)
		if file == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))
		w.Write(file.Data)
	})

	fmt.Printf("npipe running on port %s\n", port)
	fmt.Println("Press Ctrl+C to stop")
	http.ListenAndServe(":"+port, nil)
}
