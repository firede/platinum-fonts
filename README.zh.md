# platinum-fonts

[English](README.md) | 简体中文

拉丁文字与东亚文字的位图字体。西文字形绘制在 [`glyphs`](glyphs) 中；东亚文字字形
以及 Platinum Mono 的西文来自 [`sources.toml`](sources.toml) 中固定版本的字体。
`cmd/build` 将它们打包为 Go 包。

![字样](docs/specimen.png)

状态：早期。1.0 之前 API 可能变化。

## 下载

字体包附在每个 [Release](https://github.com/firede/platinum-fonts/releases) 中，每种格式一个 zip：

| 格式 | 用途 |
|---|---|
| `otf`、`ttf` | macOS、Windows 及大多数应用 |
| `woff2` | 网页 |
| `otb` | Linux 桌面（GTK、Pango） |
| `bdf`、`pcf` | X11 应用、终端 |

字体名：Platinum Sans SC、TC、JP、KR，Platinum Sans Small SC、TC、JP、KR 与
Platinum Mono SC、TC、JP、KR，各含 Regular 和 Bold。
后缀决定东亚文字字形的地区写法。em 为 12px：请在 12px 或其整数倍下使用。

## 字族

| 字族 | 包 | 样式 | 上行 | 下行 | 大写高度 | 小写高度 |
|---|---|---|---|---|---|---|
| Platinum Sans | `sans` | Regular、Bold | 9 | 3 | 9 | 6 |
| Platinum Sans Small | `sanssmall` | Regular、Bold | 8 | 3 | 8 | 5 |
| Platinum Mono | `mono` | Regular、Bold | 10 | 2 | 8 | 6 |
| 东亚文字 12px | `cjk12/zhhans`、`cjk12/zhhant`、`cjk12/ja`、`cjk12/ko` | Regular、Bold | 13 | 3 | | |

上行与下行为 `Face.Ascent` 与 `Face.Descent` 的返回值。

- Platinum Sans 与 Platinum Sans Small：比例宽度的拉丁字母，含 Latin-1 与常用中欧
  字母、排版标点，以及符号 ⌘ ↑ ↵ ⌫（仅 Platinum Sans）。
- Platinum Sans Small Bold 与 `cjk12` 的 Bold 由 Regular 算法加粗得到：每个字形向右
  1 像素再绘制一次，宽度加 1 像素。`cjk12` 的 Bold 保留笔画之间 1 像素的空隙
  （`bitmap.EmboldenOpen`），在首次使用时计算。
- `cjk12`：同一批字符的四套地区字形；每个包约 34,600 个字形，635 KB。汉字墨迹
  11×11，从基线下 1 行到基线上 10 行；步进 12。只含东亚文字区块。各字符集覆盖率：
  [docs/coverage.md](docs/coverage.md)。
- Platinum Mono：用于终端与代码，格子宽 6 像素、高 12 像素。字符占的格数由
  `mono.Cells` 给出：按 `wcwidth` 的规则取东亚宽度属性，Ambiguous 为 1 格；字形的
  步进与之相同，放不进格子的字形不收录。西文与符号取自缝合像素字体 12px 等宽版；
  它画成 2 格宽或没有收录的约一百个常用符号（如 — → ■ ✓ ≠）在
  [`glyphs/mono.txt`](glyphs/mono.txt) 中画成 1 格宽。
  框线、方块、盲文和 Powerline 分隔符由程序生成，填满格子并与相邻格子相接。东亚
  文字字形取自 `cjk12`，由 `mono.EastAsian` 下移 1 行放进格子。Bold 在格子内加粗，
  保留 1 像素的笔画空隙和字符之间的那一列；程序生成的字形两种样式相同。

## 用法

```go
import (
	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/cjk12/zhhans"
	"github.com/firede/platinum-fonts/mono"
	"github.com/firede/platinum-fonts/sans"
)

text := bitmap.Fallback(sans.Regular(), zhhans.Regular())
bold := bitmap.Fallback(sans.Bold(), zhhans.Bold())

g, ok := text.Glyph('中')                   // g.Advance == 12，g.Bounds == (0,-10)-(11,1)
end := bitmap.Draw(img, dot, text, "Hello，世界", color.Black)
w := bitmap.Measure(bold, "Hello")

term := bitmap.Fallback(mono.Regular(), mono.EastAsian(zhhans.Regular()))
n := mono.Cells('中')                       // 2：其字形步进 2 * mono.CellWidth
```

- 坐标以基线上的原点为基准，y 轴向下。
- `Glyph.Mask` 以相同坐标返回字形的 `*image.Alpha`。

## 计划

Platinum Mono 还将加入：

- AKR Pixel 连字，在 OpenType 中通过 `calt` 实现：连字的每个字符替换为连字在它那
  一格的切片，每个字形仍只前进 1 格。不需要连字时关闭 `calt` 即可；BDF、PCF 和 OTB
  不含连字。
- Nerd Fonts 图标，栅格化到网格上：步进 1 格，墨迹最宽 2 格，与 Nerd Font（而非
  Nerd Font Mono）相同，后面需要跟一个空格。常用图标手工修正。

## 构建

```sh
go run ./cmd/build          # 从 glyphs/ 与来源重新生成
go run ./cmd/build -check   # 仓库内容过期时以非零状态退出
go test ./...               # -update 重写 */testdata 中的 golden 图
```

来源下载到 `cache/`，只下载一次，并按固定的 SHA-256 校验。输出是确定的。

按发布流程生成字体包（Python 3.13，依赖见 `release/requirements.txt`）：

```sh
go run ./cmd/export -o dist/bdf
python release/build.py --version 0.1.0 --date 2026-10-15 dist/bdf dist
```

推送 `vX.Y.Z` 形式的 tag 时，`.github/workflows/release.yml` 执行相同的步骤，并把
zip 附到对应的 Release。

生成的文件：`sans/*`、`sanssmall/*`、`cjk12/*/*` 与 `mono/*`（测试、`testdata` 与
`mono/cells.go` 除外）、
`docs/coverage.md`、`docs/specimen.png`、`licenses/`、`NOTICE.md`、`OFL.txt`。

`glyphs/*.txt` 的格式见 [`internal/drawn`](internal/drawn/drawn.go)。

## 许可

- 字形数据与字体：SIL Open Font License 1.1（[OFL.txt](OFL.txt)）。来源及其许可见
  [NOTICE.md](NOTICE.md) 和 [licenses](licenses)。
- 代码：MIT（[LICENSE](LICENSE)）。
