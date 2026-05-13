package main

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"sync"
	"time"
)

var (
	private10  = mustParseCIDR("10.0.0.0/8")
	private172 = mustParseCIDR("172.16.0.0/12")
	private192 = mustParseCIDR("192.168.0.0/16")
)

func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

func isPrivateIP(ip net.IP) bool {
	return private10.Contains(ip) || private172.Contains(ip) || private192.Contains(ip)
}

func toNetwork(ip net.IP) *net.IPNet {
	switch {
	case private10.Contains(ip):
		return private10
	case private172.Contains(ip):
		return private172
	case private192.Contains(ip):
		return private192
	default:
		// Public IP: /32 for IPv4, /128 for IPv6
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
	}
}

// how often the store checks if any files expired
const storeTick = 30 * time.Second

// when a file expires and is deleted
const fileTTL = 5 * time.Minute

type File struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Data      []byte    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type FileStore struct {
	mu    sync.RWMutex
	files map[string]map[string]*File // network CIDR -> id -> file
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

func (s *FileStore) Add(ip net.IP, name string, data []byte) string {
	network := toNetwork(ip).String()
	id := make([]byte, 8)
	rand.Read(id)
	fileID := hex.EncodeToString(id)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.files[network] == nil {
		s.files[network] = make(map[string]*File)
	}
	now := time.Now()
	s.files[network][fileID] = &File{ID: fileID, Name: name, Data: data, CreatedAt: now, ExpiresAt: now.Add(fileTTL)}
	return fileID
}

func (s *FileStore) List(ip net.IP) []*File {
	network := toNetwork(ip).String()
	s.mu.RLock()
	defer s.mu.RUnlock()

	var files []*File
	for _, f := range s.files[network] {
		files = append(files, f)
	}
	return files
}

func (s *FileStore) Get(ip net.IP, id string) *File {
	network := toNetwork(ip).String()
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.files[network] == nil {
		return nil
	}
	return s.files[network][id]
}

func (s *FileStore) Delete(ip net.IP, id string) bool {
	network := toNetwork(ip).String()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.files[network] == nil {
		return false
	}
	_, exists := s.files[network][id]
	delete(s.files[network], id)
	return exists
}
