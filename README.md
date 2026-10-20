# platinum-fonts

English | [简体中文](README.zh.md)

Bitmap faces for Latin and East Asian scripts. Latin faces are drawn in
[`glyphs`](glyphs); East Asian glyphs, and the Latin of Platinum Mono, come
from fonts pinned in [`sources.toml`](sources.toml). `cmd/build` packs them
into Go packages.

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

Font names: Platinum Sans SC, TC, JP, KR, Platinum Sans Small SC, TC,
JP, KR and Platinum Mono SC, TC, JP, KR, each in Regular and Bold. The
suffix selects the regional forms of the East Asian glyphs. The em is
12px: use the fonts at 12px or a multiple of it.

## Families

| Family | Package | Styles | Ascent | Descent | Cap height | x-height |
|---|---|---|---|---|---|---|
| Platinum Sans | `sans` | Regular, Bold | 9 | 3 | 9 | 6 |
| Platinum Sans Small | `sanssmall` | Regular, Bold | 8 | 3 | 8 | 5 |
| Platinum Mono | `mono` | Regular, Bold | 10 | 2 | 8 | 6 |
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
- Platinum Mono: for terminals and code, on cells 6 pixels wide and 12
  tall. A character takes the cells `mono.Cells` reports, East Asian Width
  as `wcwidth` computes it, Ambiguous 1 cell; its glyph advances as far,
  and glyphs too wide for their cells are left out. Latin and symbols are
  Fusion Pixel 12px monospaced's; about a hundred frequent symbols it
  draws 2 cells wide or lacks, as — → ■ ✓ ≠, are drawn in
  [`glyphs/mono.txt`](glyphs/mono.txt). Box drawing, blocks, braille and
  Powerline separators are generated to fill their cells and meet their
  neighbours'. East Asian glyphs are `cjk12`'s, set a row lower by
  `mono.EastAsian`. Bold is emboldened within the cells, gaps of 1 pixel
  kept open; the generated glyphs are the same in both styles.

## Usage

```go
import (
	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/cjk12/zhhans"
	"github.com/firede/platinum-fonts/mono"
	"github.com/firede/platinum-fonts/sans"
)

text := bitmap.Fallback(sans.Regular(), zhhans.Regular())
bold := bitmap.Fallback(sans.Bold(), zhhans.Bold())

g, ok := text.Glyph('中')                   // g.Advance == 12, g.Bounds == (0,-10)-(11,1)
end := bitmap.Draw(img, dot, text, "Hello，世界", color.Black)
w := bitmap.Measure(bold, "Hello")

term := bitmap.Fallback(mono.Regular(), mono.EastAsian(zhhans.Regular()))
n := mono.Cells('中')                       // 2: its glyph advances 2 * mono.CellWidth
```

- Coordinates are relative to the dot on the baseline; y grows downward.
- `Glyph.Mask` returns a glyph as an `*image.Alpha` in the same
  coordinates.

## Planned

For Platinum Mono:

- Ligatures from AKR Pixel, in OpenType by `calt`: each character of a
  ligature becomes its cell's slice of it, so that every glyph still
  advances one cell. Turn `calt` off where ligatures are not wanted; BDF,
  PCF and OTB have none.
- Nerd Fonts icons, rasterized to the grid: each advances 1 cell, its ink
  up to 2 cells wide, as in Nerd Font (not Nerd Font Mono), and wants a
  space after it. Frequently used icons are corrected by hand.

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

Generated files: `sans/*`, `sanssmall/*`, `cjk12/*/*` and `mono/*` except
tests, `testdata` and `mono/cells.go`; `docs/coverage.md`,
`docs/specimen.png`, `licenses/`, `NOTICE.md`, `OFL.txt`.

The format of `glyphs/*.txt` is described in
[`internal/drawn`](internal/drawn/drawn.go).

## License

- Glyph data and faces: SIL Open Font License 1.1 ([OFL.txt](OFL.txt)).
  Sources and their licenses are listed in [NOTICE.md](NOTICE.md) and
  [licenses](licenses).
- Code: MIT ([LICENSE](LICENSE)).
