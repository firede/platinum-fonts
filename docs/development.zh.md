# 开发

[English](development.md) | 简体中文

## Go 包

每套字形同时也是一个 Go 包，可以把文字绘制到 `image/draw` 的图像中。这些包只依赖
标准库。

```sh
go get github.com/firede/platinum-fonts
```

| 包 | 内容 |
|---|---|
| `bitmap` | 字形、字体与后备字体；绘制和测量文字 |
| `sans`、`sanssmall` | Platinum Sans 与 Platinum Sans Small，Regular 和 Bold |
| `mono` | Platinum Mono，Regular 和 Bold，以及字符格宽和连字 |
| `cjk12/zhhans`、`cjk12/zhhant`、`cjk12/ja`、`cjk12/ko` | SC、TC、JP、KR 字形的东亚文字 |

西文字体不含东亚文字字形，需要通过 `bitmap.Fallback` 与 `cjk12` 的包组合使用：

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

g, ok := text.Glyph('中') // g.Advance == 12，g.Bounds == (0,-10)-(11,1)
```

坐标以基线上的原点为基准，y 轴向下。`Glyph.Mask` 以相同的坐标返回字形的
`*image.Alpha`。

Platinum Mono 通过 `mono.EastAsian` 把东亚文字放进格子，用 `mono.Draw` 绘制连字：

```go
term := bitmap.Fallback(mono.Regular(), mono.EastAsian(zhhans.Regular()))
n := mono.Cells('中') // 2：字形步进 2 * mono.CellWidth
end = mono.Draw(img, dot, term, mono.Ligatures(), "a != b", color.Black)
```

`cjk12` 的 Bold 在首次使用时生成。1.0 之前 API 可能变化。

## 仓库结构

| 路径 | 内容 |
|---|---|
| `glyphs/` | 本仓库绘制的字形，以文本存储 |
| `sources.toml` | 其他字形的来源字体，按 SHA-256 固定版本 |
| `cmd/build` | 生成 Go 包、文档和版权声明 |
| `cmd/export` | 把字体导出为 BDF，供 `release/build.py` 使用 |
| `release/` | 制作发布用的字体包 |
| `internal/` | 字形格式、Mono 的生成字形、连字和字符格宽 |

`glyphs/*.txt` 的绘制格式见 [`internal/drawn`](../internal/drawn/drawn.go)。

## 构建

```sh
go run ./cmd/build          # 从 glyphs/ 和来源字体重新生成
go run ./cmd/build -check   # 生成的文件过期时报错
go test ./...               # go test ./... -update 重写 golden 图
```

来源字体只下载一次，存放在 `cache/` 中，并按 SHA-256 校验。输出是确定的。

以下文件是生成的，请修改 `glyphs/` 或 `cmd/build`，不要直接编辑：

- `sans/`、`sanssmall/`、`cjk12/*/` 和 `mono/`，测试、`testdata`、`mono/cells.go`
  与 `mono/ligatures.go` 除外
- `docs/coverage.md`、`docs/specimen.png`、`docs/mono.png`、
  `docs/social.png`
- `licenses/`、`NOTICE.md`、`OFL.txt`

## 发布

制作字体包需要 Python 3.13 和 `release/requirements.txt` 中的依赖：

```sh
go run ./cmd/export -o dist/bdf
python release/build.py --version 0.3.0 --date 2026-10-21 dist/bdf dist
```

这会在 `dist` 中为每种格式生成一个 zip。推送 `vX.Y.Z` 形式的 tag 时，
[`release.yml`](../.github/workflows/release.yml) 会执行相同的步骤，把 zip 附到
Release 上，并以 [`release/notes.md`](../release/notes.md) 作为说明。
