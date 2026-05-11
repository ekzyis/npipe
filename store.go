package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// how often the store checks if any files expired
const storeTick = 30 * time.Second

// when a file expires and is deleted
const fileTTL = 5 * time.Minute

type File struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Data      []byte    `json:"-"`
	CreatedAt time.Time `json:"-"`
}

type FileStore struct {
	mu    sync.RWMutex
	files map[string]map[string]*File // ip -> id -> file
}

func NewFileStore() *FileStore {
	store := &FileStore{files: make(map[string]map[string]*File)}
	go store.routinelyDeleteExpired()
	return store
}

func (s *FileStore) routinelyDeleteExpired() {
	ticker := time.NewTicker(storeTick)
	for range ticker.C {
		s.deletedExpired()
	}
}

func (s *FileStore) deletedExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for ip, files := range s.files {
		for id, file := range files {
			if now.Sub(file.CreatedAt) > fileTTL {
				delete(files, id)
			}
		}
		if len(files) == 0 {
			delete(s.files, ip)
		}
	}
}

func (s *FileStore) Add(ip, name string, data []byte) string {
	id := make([]byte, 8)
	rand.Read(id)
	fileID := hex.EncodeToString(id)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.files[ip] == nil {
		s.files[ip] = make(map[string]*File)
	}
	s.files[ip][fileID] = &File{ID: fileID, Name: name, Data: data, CreatedAt: time.Now()}
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

func (s *FileStore) Delete(ip, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.files[ip] == nil {
		return false
	}
	_, exists := s.files[ip][id]
	delete(s.files[ip], id)
	return exists
}
