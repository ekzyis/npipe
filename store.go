package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

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
