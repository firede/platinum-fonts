# Platinum Fonts

English | [简体中文](README.zh.md)

Pixel fonts for interfaces and terminals, drawn on a 12-pixel grid. They
set Latin text alongside Chinese, Japanese and Korean, and stay sharp at
12 pixels and its multiples.

![Platinum Sans, Platinum Sans Small and Platinum Mono, with Chinese, Japanese and Korean](docs/specimen.png)

## Families

- **Platinum Sans**: a proportional sans for interface text.
- **Platinum Sans Small**: a smaller sans for dense layouts, such as
  toolbars and status bars.
- **Platinum Mono**: a monospaced font for terminals and code. Chinese,
  Japanese and Korean take two cells, aligned with the Latin.
  Programming ligatures and the icons of Nerd Fonts are built in.

Each family comes in Regular and Bold, and in four regional versions:
**SC** (Simplified Chinese), **TC** (Traditional Chinese), **JP**
(Japanese) and **KR** (Korean). They share their Latin glyphs; the
version decides the regional forms of the Han characters. Pick the one
for the language you read most.

![Platinum Mono in a terminal: a directory listed with icons, code with ligatures and Chinese, and a system monitor](docs/mono.png)

## Install

Download a zip from [Releases](https://github.com/firede/platinum-fonts/releases).
Each format comes in its own zip:

| Format | For |
|---|---|
| `otf` | macOS, Windows and most applications |
| `ttf` | Applications that do not take OTF |
| `woff2` | Web pages |
| `otb` | Linux desktops (GTK, Pango) |
| `bdf`, `pcf` | X11 applications |

- **macOS**: open the font files and click Install Font.
- **Windows**: select the font files, right-click, and choose Install.
- **Linux**: copy the font files to `~/.local/share/fonts` and run
  `fc-cache -f`.

## Use them at 12 pixels

Each glyph is drawn pixel by pixel on a 12-pixel grid, so the fonts are
sharp only at 12 pixels or a multiple of it (24, 36…). At other sizes they
blur.

| Where | Size |
|---|---|
| macOS (Retina or not) | 12 or 24 |
| Windows at 100% or 200% scaling | 9 pt or 18 pt |
| Web pages | `font-size: 12px` or `24px` |

On Windows at other scalings, such as 125% or 150%, no point size lands on
whole pixels.

## Platinum Mono in a terminal

Set the font family to `Platinum Mono SC` (or TC, JP, KR) and the size to
12.

- **Line height** is built in, so leave your terminal's line height at its
  default. A cell is 7 pixels wide and 14 tall.
- **Icons** draw two cells wide but advance one, as in the Nerd Font
  fonts (not Nerd Font Mono). Leave a space after each; tools such as
  eza and Starship already do.
- **Ligatures** join `=>` `!=` `<=` `->` `::` `//` and others, 31 in all.
  They are on by default, through the OpenType feature `calt`; BDF and
  PCF have none. To turn them off:

  | Application | Setting |
  |---|---|
  | Ghostty | `font-feature = -calt` |
  | Zed | `"buffer_font_features": { "calt": false }` |
  | VS Code | `"editor.fontLigatures": false` |
  | Kitty | `disable_ligatures always` |
  | WezTerm | `harfbuzz_features = { "calt=0" }` |

- **Ambiguous-width characters**, such as `○` `★` and Greek and Cyrillic
  letters, take one cell. Keep your terminal's ambiguous width set to
  narrow, the usual default.

## For developers

The fonts are also Go packages that draw text into images; see
[Development](docs/development.md). How each family is made, with its
metrics and sources, is described in [Design](docs/design.md).

## License

The fonts are under the [SIL Open Font License 1.1](OFL.txt). Platinum
Mono also contains ligatures under the MIT License and icons under the
licenses of their icon sets; [NOTICE.md](NOTICE.md) lists every source
and its license. The code is under the [MIT License](LICENSE).
