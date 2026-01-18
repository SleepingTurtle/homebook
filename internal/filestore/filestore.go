package filestore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Store handles local file storage
type Store struct {
	basePath string
}

// New creates a new file store with the given base path
func New(basePath string) (*Store, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create filestore directory: %w", err)
	}
	return &Store{basePath: basePath}, nil
}

// Save stores a file and returns the relative path
func (s *Store) Save(filename string, r io.Reader) (string, error) {
	// Generate unique filename to avoid collisions
	uniqueID, err := generateID()
	if err != nil {
		return "", fmt.Errorf("generate file id: %w", err)
	}

	// Preserve original extension
	ext := filepath.Ext(filename)
	newFilename := uniqueID + ext

	// Create full path
	fullPath := filepath.Join(s.basePath, newFilename)

	// Create file
	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	// Copy data to file
	if _, err := io.Copy(f, r); err != nil {
		os.Remove(fullPath) // Clean up on error
		return "", fmt.Errorf("write file: %w", err)
	}

	return newFilename, nil
}

// Get returns a reader for the file at the given path
func (s *Store) Get(filename string) (*os.File, error) {
	fullPath := filepath.Join(s.basePath, filename)
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

// Delete removes the file at the given path
func (s *Store) Delete(filename string) error {
	if filename == "" {
		return nil
	}
	fullPath := filepath.Join(s.basePath, filename)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

// FullPath returns the full filesystem path for a filename
func (s *Store) FullPath(filename string) string {
	return filepath.Join(s.basePath, filename)
}

// generateID creates a random 16-character hex string
func generateID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
