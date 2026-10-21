# Development

English | [简体中文](development.zh.md)

## Go packages

Each face is also a Go package that draws text into an `image/draw` image.
The packages depend only on the standard library.

```sh
go get github.com/firede/platinum-fonts
```

| Package | Contents |
|---|---|
| `bitmap` | Glyphs, faces and fallback; drawing and measuring text |
| `sans`, `sanssmall` | Platinum Sans and Platinum Sans Small, Regular and Bold |
| `mono` | Platinum Mono, Regular and Bold, with its cell widths and ligatures |
| `cjk12/zhhans`, `cjk12/zhhant`, `cjk12/ja`, `cjk12/ko` | East Asian glyphs in the forms of SC, TC, JP and KR |

The Latin faces hold no East Asian glyphs. Combine them with a `cjk12`
package through `bitmap.Fallback`:

```go
import (
	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/cjk12/zhhans"
	"github.com/firede/platinum-fonts/mono"
	"github.com/firede/platinum-fonts/sans"
)

text := bitmap.Fallback(sans.Regular(), zhhans.Regular())
end := bitmap.Draw(img, dot, text, "Hello，世界", color.Black)
w := bitmap.Measure(text, "Hello")

g, ok := text.Glyph('中') // g.Advance == 12, g.Bounds == (0,-10)-(11,1)
```

Coordinates are relative to the dot on the baseline, with y growing
downward. `Glyph.Mask` returns a glyph as an `*image.Alpha` in the same
coordinates.

Platinum Mono sets East Asian glyphs in its cells through
`mono.EastAsian`, and draws its ligatures with `mono.Draw`:

```go
term := bitmap.Fallback(mono.Regular(), mono.EastAsian(zhhans.Regular()))
n := mono.Cells('中') // 2: the glyph advances 2 * mono.CellWidth
end = mono.Draw(img, dot, term, mono.Ligatures(), "a != b", color.Black)
```

The Bold faces of `cjk12` are made on first use. The API may change
before 1.0.

## Repository

| Path | Contents |
|---|---|
| `glyphs/` | Glyphs drawn for this repository, in text |
| `sources.toml` | The fonts the other glyphs come from, pinned by SHA-256 |
| `cmd/build` | Builds the Go packages, the docs and the notices |
| `cmd/export` | Writes the fonts as BDF, for `release/build.py` |
| `release/` | Makes the font packages of a release |
| `internal/` | The glyph format, the generated Mono glyphs, ligatures and cell widths |

The drawing format of `glyphs/*.txt` is described in
[`internal/drawn`](../internal/drawn/drawn.go).

## Build

```sh
go run ./cmd/build          # regenerate everything from glyphs/ and the sources
go run ./cmd/build -check   # fail if a generated file is out of date
go test ./...               # go test ./... -update rewrites the golden images
```

Sources are downloaded once into `cache/` and verified against their
SHA-256. The output is deterministic.

These files are generated; edit `glyphs/` or `cmd/build` instead:

- `sans/`, `sanssmall/`, `cjk12/*/` and `mono/`, except the tests,
  `testdata`, `mono/cells.go` and `mono/ligatures.go`
- `docs/coverage.md`, `docs/specimen.png`, `docs/mono.png`
- `licenses/`, `NOTICE.md`, `OFL.txt`

## Release

The font packages need Python 3.13 and `release/requirements.txt`:

```sh
go run ./cmd/export -o dist/bdf
python release/build.py --version 0.3.0 --date 2026-10-21 dist/bdf dist
```

This writes one zip per format into `dist`. Pushing a tag `vX.Y.Z` runs
the same steps in [`release.yml`](../.github/workflows/release.yml) and
attaches the zips to the release, with [`release/notes.md`](../release/notes.md)
as its description.
