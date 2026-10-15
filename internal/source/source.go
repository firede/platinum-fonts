// Package source fetches the fonts sources.toml pins: it downloads each
// once into a cache, checks its SHA-256, and reads the files of an archive.
package source

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Source is a pinned artifact.
type Source struct {
	Name       string `toml:"-"`
	Repository string `toml:"repository"`
	Version    string `toml:"version"`
	Commit     string `toml:"commit"`
	URL        string `toml:"url"`
	SHA256     string `toml:"sha256"`
	License    string `toml:"license"`
}

// Pinned returns the version or the commit the source is pinned to.
func (s *Source) Pinned() string {
	if s.Version != "" {
		return s.Version
	}
	return s.Commit
}

// Load reads the sources of a manifest.
func Load(manifest string) (map[string]*Source, error) {
	var m map[string]*Source
	if _, err := toml.DecodeFile(manifest, &m); err != nil {
		return nil, err
	}
	for name, s := range m {
		s.Name = name
		if s.URL == "" || len(s.SHA256) != 64 {
			return nil, fmt.Errorf("%s: source %s needs a url and a sha256", manifest, name)
		}
	}
	return m, nil
}

// Fetch returns the contents of s, downloaded into cache unless it is
// already there. It fails if they are not what s pins.
func Fetch(s *Source, cache string) ([]byte, error) {
	file := filepath.Join(cache, s.SHA256[:12]+"-"+path.Base(s.URL))
	data, err := os.ReadFile(file)
	if err == nil && sum(data) == s.SHA256 {
		return data, nil
	}
	resp, err := http.Get(s.URL)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", s.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s: %s", s.Name, s.URL, resp.Status)
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", s.Name, err)
	}
	if got := sum(data); got != s.SHA256 {
		return nil, fmt.Errorf("%s: %s has SHA-256 %s, not the %s pinned", s.Name, s.URL, got, s.SHA256)
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return nil, err
	}
	return data, os.WriteFile(file, data, 0o644)
}

func sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// Archive is a zip archive of a source.
type Archive struct{ r *zip.Reader }

// OpenArchive reads data as a zip archive.
func OpenArchive(data []byte) (*Archive, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	return &Archive{r}, nil
}

// File returns the contents of the file at name in the archive.
func (a *Archive) File(name string) ([]byte, error) {
	f, err := a.r.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
