Pixel fonts for interfaces and terminals, drawn on a 12-pixel grid:

- **Platinum Sans**: a proportional sans for interface text.
- **Platinum Sans Small**: a smaller sans for dense layouts.
- **Platinum Mono**: a monospaced font for terminals and code, with
  programming ligatures and the icons of Nerd Fonts.

Each comes in Regular and Bold, in four regional versions: SC, TC, JP and
KR. Use them at 12 pixels or a multiple of it; in a terminal, set
Platinum Mono to size 12. See the [README](https://github.com/firede/platinum-fonts#readme)
for details.

Download one zip per format:

| Format | For |
|---|---|
| `otf` | macOS, Windows and most applications |
| `ttf` | Applications that do not take OTF |
| `woff2` | Web pages |
| `otb` | Linux desktops (GTK, Pango) |
| `bdf`, `pcf` | X11 applications |

Each zip holds every family and region, with OFL.txt, NOTICE.md and the
licenses of the sources.

Go packages of this version: `go get github.com/firede/platinum-fonts@<tag>`.
