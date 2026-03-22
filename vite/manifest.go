package vite

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
)

type Manifest struct {
	entries map[string]*ManifestEntry
	mu      sync.RWMutex
}

type ManifestEntry struct {
	File    string            `json:"file"`
	Src     string            `json:"src,omitempty"`
	Imports []string          `json:"imports,omitempty"`
	CSS     []string          `json:"css,omitempty"`
	Assets  []string          `json:"assets,omitempty"`
	IsEntry bool              `json:"isEntry,omitempty"`
	Others  map[string]string `json:"-"`
}

type Loader struct {
	path   string
	dev    bool
	fs     fs.FS
	cached atomic.Value
}

func NewLoader(path string, dev bool) *Loader {
	return &Loader{path: path, dev: dev}
}

func NewLoaderWithFS(path string, dev bool, fsys fs.FS) *Loader {
	return &Loader{path: path, dev: dev, fs: fsys}
}

func (l *Loader) Asset(entry string) (string, error) {
	var manifest *Manifest

	if l.dev {
		m, err := l.loadManifest()
		if err != nil {
			return "http://localhost:5173/" + entry, nil
		}
		manifest = m
	} else {
		if cached := l.cached.Load(); cached != nil {
			manifest = cached.(*Manifest)
		} else {
			m, err := l.loadManifest()
			if err != nil {
				return "", err
			}
			l.cached.Store(m)
			manifest = m
		}
	}

	resolved, err := manifest.Resolve(entry)
	if err != nil {
		return "", err
	}

	return "/build/" + resolved, nil
}

func New() *Manifest {
	return &Manifest{entries: make(map[string]*ManifestEntry)}
}

func (l *Loader) loadManifest() (*Manifest, error) {
	if l.fs != nil {
		return LoadFromFS(l.fs, l.path)
	}
	return Load(l.path)
}

func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	return Parse(data)
}

func LoadFromFS(fsys fs.FS, path string) (*Manifest, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file from fs: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (*Manifest, error) {
	m := New()

	var rawEntries map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawEntries); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for key, rawMsg := range rawEntries {
		entry := &ManifestEntry{Others: make(map[string]string)}
		if err := json.Unmarshal(rawMsg, entry); err != nil {
			return nil, fmt.Errorf("failed to parse entry %s: %w", key, err)
		}
		m.entries[key] = entry
	}

	return m, nil
}

func (m *Manifest) Resolve(path string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.entries[path]; ok {
		return entry.File, nil
	}

	if len(path) > 0 {
		if entry, ok := m.entries[path[1:]]; ok {
			return entry.File, nil
		}
	}

	return "", fmt.Errorf("asset not found in manifest: %s", path)
}
