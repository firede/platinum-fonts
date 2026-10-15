# platinum-fonts

English | [简体中文](README.zh.md)

Bitmap faces for Latin and East Asian scripts. Latin faces are drawn in
[`glyphs`](glyphs); East Asian glyphs come from fonts pinned in
[`sources.toml`](sources.toml). `cmd/build` packs both into Go packages.

![Specimen](docs/specimen.png)

Status: early. The API may change before 1.0.

## Download

Font packages are attached to each [release](https://github.com/firede/platinum-fonts/releases),
one zip per format:

| Format | For |
|---|---|
| `otf`, `ttf` | macOS, Windows, most applications |
| `woff2` | Web pages |
| `otb` | Linux desktops (GTK, Pango) |
| `bdf`, `pcf` | X11 applications, terminals |

Font names: Platinum Sans SC, TC, JP, KR and Platinum Sans Small SC, TC,
JP, KR, each in Regular and Bold. The suffix selects the regional forms
of the East Asian glyphs. The em is 12px: use the fonts at 12px or a
multiple of it.

## Families

| Family | Package | Styles | Ascent | Descent | Cap height | x-height |
|---|---|---|---|---|---|---|
| Platinum Sans | `sans` | Regular, Bold | 9 | 3 | 9 | 6 |
| Platinum Sans Small | `sanssmall` | Regular, Bold | 8 | 3 | 8 | 5 |
| East Asian, 12px | `cjk12/zhhans`, `cjk12/zhhant`, `cjk12/ja`, `cjk12/ko` | Regular, Bold | 13 | 3 | | |

Ascent and descent are those `Face.Ascent` and `Face.Descent` return.

- Platinum Sans and Platinum Sans Small: proportional Latin, Latin-1 and
  common Central European letters, typographic punctuation, and the
  symbols ⌘ ↑ ↵ ⌫ (Platinum Sans only).
- Platinum Sans Small Bold and the `cjk12` Bold faces are Regular
  emboldened: each glyph drawn again 1 pixel to the right, 1 pixel wider.
  `cjk12` Bold keeps gaps of 1 pixel between strokes open
  (`bitmap.EmboldenOpen`), and is computed on first use.
- `cjk12`: the same characters in four sets of regional forms; about
  34,600 glyphs and 635 KB per package. Ideographs: 11×11 ink, from 1 row
  below the baseline to 10 rows above it; advance 12. East Asian blocks
  only. Coverage by national character set: [docs/coverage.md](docs/coverage.md).

## Usage

```go
import (
	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/cjk12/zhhans"
	"github.com/firede/platinum-fonts/sans"
)

text := bitmap.Fallback(sans.Regular(), zhhans.Regular())
bold := bitmap.Fallback(sans.Bold(), zhhans.Bold())

g, ok := text.Glyph('中')                   // g.Advance == 12, g.Bounds == (0,-10)-(11,1)
end := bitmap.Draw(img, dot, text, "Hello，世界", color.Black)
w := bitmap.Measure(bold, "Hello")
```

- Coordinates are relative to the dot on the baseline; y grows downward.
- `Glyph.Mask` returns a glyph as an `*image.Alpha` in the same
  coordinates.

## Planned

Platinum Mono, for terminals and code:

| Cell | Latin | East Asian | Other |
|---|---|---|---|
| 6×12, ascent 10 | Fusion Pixel 12px monospaced | Fusion Pixel 12px monospaced, 2 cells | Ligatures from AKR Pixel; Nerd Fonts icons; procedural box drawing, blocks, braille and Powerline glyphs |

- Cell width follows Unicode East Asian Width as `wcwidth` computes it;
  Ambiguous is 1 cell. Source advances are not used.
- U+2500–257F, U+2580–259F, U+2800–28FF and Powerline separators are
  generated to fill their cells.
- A ligature of N characters is one glyph N cells wide.
- Nerd Fonts icons are rasterized to the grid; frequently used icons are
  corrected by hand.

## Build

```sh
go run ./cmd/build          # regenerate from glyphs/ and the sources
go run ./cmd/build -check   # exit non-zero if the repository is out of date
go test ./...               # -update rewrites the golden images in */testdata
```

Sources are downloaded once into `cache/` and verified against their
pinned SHA-256. Output is deterministic.

Font packages, as a release makes them (Python 3.13, `release/requirements.txt`):

```sh
go run ./cmd/export -o dist/bdf
python release/build.py --version 0.1.0 --date 2026-10-15 dist/bdf dist
```

Pushing a tag `vX.Y.Z` runs the same steps in `.github/workflows/release.yml`
and attaches the zips to the release.

Generated files: `sans/*`, `sanssmall/*` and `cjk12/*/*` except tests and
`testdata`, `docs/coverage.md`, `docs/specimen.png`, `licenses/`,
`NOTICE.md`, `OFL.txt`.

The format of `glyphs/*.txt` is described in
[`internal/drawn`](internal/drawn/drawn.go).

## License

- Glyph data and faces: SIL Open Font License 1.1 ([OFL.txt](OFL.txt)).
  Sources and their licenses are listed in [NOTICE.md](NOTICE.md) and
  [licenses](licenses).
- Code: MIT ([LICENSE](LICENSE)).
