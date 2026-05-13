package main

import (
	"net"
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add(net.ParseIP("1.2.3.4"), "test.txt", []byte("hello"))

	if id == "" {
		t.Fatal("expected non-empty id")
	}
	if len(id) != 16 {
		t.Fatalf("expected 16 char hex id, got %s", id)
	}
}

func TestGet(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add(net.ParseIP("1.2.3.4"), "test.txt", []byte("hello"))

	file := store.Get(net.ParseIP("1.2.3.4"), id)
	if file == nil {
		t.Fatal("expected file, got nil")
	}
	if file.Name != "test.txt" {
		t.Fatalf("expected name test.txt, got %s", file.Name)
	}
	if string(file.Data) != "hello" {
		t.Fatalf("expected data hello, got %s", file.Data)
	}
}

func TestGetWrongIP(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add(net.ParseIP("1.2.3.4"), "test.txt", []byte("hello"))

	file := store.Get(net.ParseIP("5.6.7.8"), id)
	if file != nil {
		t.Fatal("expected nil for different IP")
	}
}

func TestGetNotFound(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}

	file := store.Get(net.ParseIP("1.2.3.4"), "nonexistent")
	if file != nil {
		t.Fatal("expected nil for nonexistent file")
	}
}

func TestList(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	store.Add(net.ParseIP("1.2.3.4"), "a.txt", []byte("a"))
	store.Add(net.ParseIP("1.2.3.4"), "b.txt", []byte("b"))
	store.Add(net.ParseIP("5.6.7.8"), "c.txt", []byte("c"))

	files := store.List(net.ParseIP("1.2.3.4"))
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	files = store.List(net.ParseIP("5.6.7.8"))
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	files = store.List(net.ParseIP("9.10.11.12"))
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}

	// Test private IPs in same network share files
	store.Add(net.ParseIP("192.168.1.1"), "d.txt", []byte("d"))
	store.Add(net.ParseIP("192.168.50.100"), "e.txt", []byte("e"))
	store.Add(net.ParseIP("10.0.0.1"), "f.txt", []byte("f"))
	store.Add(net.ParseIP("172.16.5.5"), "g.txt", []byte("g"))

	// All 192.168.x.x IPs share the same network
	files = store.List(net.ParseIP("192.168.255.255"))
	if len(files) != 2 {
		t.Fatalf("expected 2 files in 192.168.0.0/16, got %d", len(files))
	}

	// 10.x.x.x is separate
	files = store.List(net.ParseIP("10.255.255.255"))
	if len(files) != 1 {
		t.Fatalf("expected 1 file in 10.0.0.0/8, got %d", len(files))
	}

	// 172.16.x.x is separate
	files = store.List(net.ParseIP("172.31.255.255"))
	if len(files) != 1 {
		t.Fatalf("expected 1 file in 172.16.0.0/12, got %d", len(files))
	}
}

func TestDelete(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add(net.ParseIP("1.2.3.4"), "test.txt", []byte("hello"))

	ok := store.Delete(net.ParseIP("1.2.3.4"), id)
	if !ok {
		t.Fatal("expected delete to succeed")
	}

	file := store.Get(net.ParseIP("1.2.3.4"), id)
	if file != nil {
		t.Fatal("expected file to be deleted")
	}
}

func TestDeleteWrongIP(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add(net.ParseIP("1.2.3.4"), "test.txt", []byte("hello"))

	ok := store.Delete(net.ParseIP("5.6.7.8"), id)
	if ok {
		t.Fatal("expected delete to fail for different IP")
	}

	file := store.Get(net.ParseIP("1.2.3.4"), id)
	if file == nil {
		t.Fatal("file should still exist")
	}
}

func TestDeleteNotFound(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}

	ok := store.Delete(net.ParseIP("1.2.3.4"), "nonexistent")
	if ok {
		t.Fatal("expected delete to fail for nonexistent file")
	}
}

func TestExpiry(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}

	// add file with old timestamp (use CIDR key directly)
	store.files["1.2.3.4/32"] = make(map[string]*File)
	store.files["1.2.3.4/32"]["oldfile"] = &File{
		ID:        "oldfile",
		Name:      "old.txt",
		Data:      []byte("old"),
		CreatedAt: time.Now().Add(-6 * time.Minute),
	}
	store.files["1.2.3.4/32"]["newfile"] = &File{
		ID:        "newfile",
		Name:      "new.txt",
		Data:      []byte("new"),
		CreatedAt: time.Now(),
	}

	store.deletedExpired()

	if store.Get(net.ParseIP("1.2.3.4"), "oldfile") != nil {
		t.Fatal("expected old file to be expired")
	}
	if store.Get(net.ParseIP("1.2.3.4"), "newfile") == nil {
		t.Fatal("expected new file to still exist")
	}
}

func TestPrivateIPSharing(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}

	// Add file from one private IP
	id := store.Add(net.ParseIP("192.168.1.1"), "test.txt", []byte("hello"))

	// Should be accessible from another private IP in same range
	file := store.Get(net.ParseIP("192.168.50.100"), id)
	if file == nil {
		t.Fatal("expected file to be shared across private IPs")
	}

	// Should be listable from any 192.168.x.x IP
	files := store.List(net.ParseIP("192.168.255.255"))
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Should be deletable from any 192.168.x.x IP
	ok := store.Delete(net.ParseIP("192.168.0.1"), id)
	if !ok {
		t.Fatal("expected delete to succeed from different private IP")
	}
}
