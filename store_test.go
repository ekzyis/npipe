package main

import (
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add("192.168.1.1", "test.txt", []byte("hello"))

	if id == "" {
		t.Fatal("expected non-empty id")
	}
	if len(id) != 8 {
		t.Fatalf("expected 8 char hex id, got %s", id)
	}
}

func TestGet(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add("192.168.1.1", "test.txt", []byte("hello"))

	file := store.Get("192.168.1.1", id)
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
	id := store.Add("192.168.1.1", "test.txt", []byte("hello"))

	file := store.Get("192.168.1.2", id)
	if file != nil {
		t.Fatal("expected nil for different IP")
	}
}

func TestGetNotFound(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}

	file := store.Get("192.168.1.1", "nonexistent")
	if file != nil {
		t.Fatal("expected nil for nonexistent file")
	}
}

func TestList(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	store.Add("192.168.1.1", "a.txt", []byte("a"))
	store.Add("192.168.1.1", "b.txt", []byte("b"))
	store.Add("192.168.1.2", "c.txt", []byte("c"))

	files := store.List("192.168.1.1")
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	files = store.List("192.168.1.2")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	files = store.List("192.168.1.3")
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestDelete(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add("192.168.1.1", "test.txt", []byte("hello"))

	ok := store.Delete("192.168.1.1", id)
	if !ok {
		t.Fatal("expected delete to succeed")
	}

	file := store.Get("192.168.1.1", id)
	if file != nil {
		t.Fatal("expected file to be deleted")
	}
}

func TestDeleteWrongIP(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}
	id := store.Add("192.168.1.1", "test.txt", []byte("hello"))

	ok := store.Delete("192.168.1.2", id)
	if ok {
		t.Fatal("expected delete to fail for different IP")
	}

	file := store.Get("192.168.1.1", id)
	if file == nil {
		t.Fatal("file should still exist")
	}
}

func TestDeleteNotFound(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}

	ok := store.Delete("192.168.1.1", "nonexistent")
	if ok {
		t.Fatal("expected delete to fail for nonexistent file")
	}
}

func TestExpiry(t *testing.T) {
	store := &FileStore{files: make(map[string]map[string]*File)}

	// add file with old timestamp
	store.files["192.168.1.1"] = make(map[string]*File)
	store.files["192.168.1.1"]["oldfile"] = &File{
		ID:        "oldfile",
		Name:      "old.txt",
		Data:      []byte("old"),
		CreatedAt: time.Now().Add(-6 * time.Minute),
	}
	store.files["192.168.1.1"]["newfile"] = &File{
		ID:        "newfile",
		Name:      "new.txt",
		Data:      []byte("new"),
		CreatedAt: time.Now(),
	}

	store.deletedExpired()

	if store.Get("192.168.1.1", "oldfile") != nil {
		t.Fatal("expected old file to be expired")
	}
	if store.Get("192.168.1.1", "newfile") == nil {
		t.Fatal("expected new file to still exist")
	}
}
