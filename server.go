package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strings"
)

//go:embed public/index.html
var indexHTML string

//go:embed public/style.css
var styleCSS string

//go:embed public/app.js
var appJS string

//go:embed public/manifest.json
var manifestJSON string

var commit = getCommit()

func getCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(out))
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

var store = NewFileStore()

func setupRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		html := strings.Replace(indexHTML, "__CLIENT_IP__", ip, 1)
		html = strings.ReplaceAll(html, "__COMMIT__", commit)
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

	http.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Write([]byte(manifestJSON))
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
		id := store.Add(ip, req.Name, req.Data)
		w.Write([]byte("OK"))
		log.Printf("%s uploaded %s (%d bytes)", ip, id, len(req.Data))
	})

	http.HandleFunc("/file/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/file/")
		ip := getClientIP(r)

		if r.Method == "DELETE" {
			if store.Delete(ip, id) {
				w.WriteHeader(http.StatusNoContent)
				log.Printf("%s deleted %s", ip, id)
			} else {
				http.NotFound(w, r)
			}
			return
		}

		file := store.Get(ip, id)
		if file == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))
		w.Write(file.Data)
		log.Printf("%s downloaded %s (%d bytes)", ip, id, len(file.Data))
	})
}

func runServer(port string) {
	setupRoutes()
	fmt.Printf("npipe running on port %s\n", port)
	fmt.Println("Press Ctrl+C to stop")
	http.ListenAndServe(":"+port, nil)
}
