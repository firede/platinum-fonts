// Command build makes the faces of this repository from the glyphs drawn
// in glyphs/ and the sources pinned in sources.toml, and writes them with
// their coverage report, specimen and notices. With -check, it writes
// nothing and fails if what it makes differs from what is in the
// repository.
//
// Run it from the repository's root:
//
//	go run ./cmd/build
//	go run ./cmd/build -check
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/firede/platinum-fonts/internal/source"
)

func main() {
	check := flag.Bool("check", false, "write nothing; fail if the repository differs from what the sources make")
	cache := flag.String("cache", "cache", "the directory to keep downloaded sources in")
	flag.Parse()
	log.SetFlags(0)

	sources, err := source.Load("sources.toml")
	if err != nil {
		log.Fatal(err)
	}
	out := map[string][]byte{}
	latin, err := buildLatin(out)
	if err != nil {
		log.Fatal(err)
	}
	cjk, err := buildCJK(sources, *cache, out)
	if err != nil {
		log.Fatal(err)
	}
	mono, err := buildMono(sources, *cache, cjk, out)
	if err != nil {
		log.Fatal(err)
	}
	if err := buildDocs(latin, cjk, mono, out); err != nil {
		log.Fatal(err)
	}

	stale := 0
	for _, name := range slices.Sorted(maps.Keys(out)) {
		old, err := os.ReadFile(name)
		same := err == nil && bytes.Equal(old, out[name])
		switch {
		case *check && !same:
			fmt.Println("out of date:", name)
			stale++
		case !*check && !same:
			if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
				log.Fatal(err)
			}
			if err := os.WriteFile(name, out[name], 0o644); err != nil {
				log.Fatal(err)
			}
			fmt.Println("wrote", name)
		}
	}
	if stale > 0 {
		log.Fatalf("%d files differ from what the sources make: run go run ./cmd/build", stale)
	}
}
