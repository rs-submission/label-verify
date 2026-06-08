package storage

import (
	"os"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	store := NewFileStore(t.TempDir())

	ref, err := store.Put("image-1.jpg", []byte("image bytes"))
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if ref != "image-1.jpg" {
		t.Fatalf("ref=%q want image-1.jpg", ref)
	}

	got, err := store.Get(ref)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(got) != "image bytes" {
		t.Fatalf("Get=%q want image bytes", string(got))
	}
}

func TestFileStoreRejectsPathTraversalRef(t *testing.T) {
	store := NewFileStore(t.TempDir())

	if _, err := store.Get("../secret"); err == nil {
		t.Fatal("expected invalid ref error")
	}
}

func TestFileStoreDeleteRemovesRef(t *testing.T) {
	store := NewFileStore(t.TempDir())

	ref, err := store.Put("image-1.jpg", []byte("image bytes"))
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if err := store.Delete(ref); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := store.Get(ref); !os.IsNotExist(err) {
		t.Fatalf("Get after Delete error=%v want not exist", err)
	}
}

func TestFileStoreDeleteIgnoresMissingRef(t *testing.T) {
	store := NewFileStore(t.TempDir())

	if err := store.Delete("missing.jpg"); err != nil {
		t.Fatalf("Delete missing ref returned error: %v", err)
	}
}
