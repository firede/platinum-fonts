# 设计

[English](design.md) | 简体中文

所有字族都绘制在 12 像素的 em 上。Platinum Sans 与 Platinum Sans Small 的西文为
本仓库绘制，存放在 [`glyphs`](../glyphs) 中；东亚文字字形以及 Platinum Mono 的
西文来自 [`sources.toml`](../sources.toml) 中固定版本的字体。各来源的署名见
[NOTICE.md](../NOTICE.md)。

## 度量

单位为像素。上行与下行是一行文字在基线上方和下方占用的行数，即 `Face.Ascent` 与
`Face.Descent` 的返回值。

| 字族 | 上行 | 下行 | 大写高度 | 小写高度 |
|---|---|---|---|---|
| Platinum Sans | 9 | 3 | 9 | 6 |
| Platinum Sans Small | 8 | 3 | 8 | 5 |
| Platinum Mono | 11 | 3 | 8 | 6 |
| 东亚文字（`cjk12`） | 13 | 3 | | |

## Platinum Sans 与 Platinum Sans Small

两者都覆盖 Latin-1、常用中欧字母和排版标点。Platinum Sans 另有符号 ⌘ ↑ ↵ ⌫。

Platinum Sans Bold 为手绘。Platinum Sans Small Bold 由 Regular 加粗得到：每个字形
向右偏移 1 像素再绘制一次，因此宽 1 像素。

带变音符号的字母由字母和符号组合而成。字形文件的格式（包括变音符号）见
[`internal/drawn`](../internal/drawn/drawn.go)。

## 东亚文字

东亚文字字形取自[缝合像素字体](https://github.com/TakWolf/fusion-pixel-font) 12px
比例宽度版，未做修改。每个地区一套：SC、TC、JP、KR。它们收录相同的约 34,600 个
字符，各按本地区的字形绘制。对各国字符集的覆盖率见[覆盖率](coverage.md)。

汉字的墨迹为 11×11 像素，从基线下 1 行到基线上 10 行，步进 12 像素。

Bold 由 Regular 加粗得到，并保留笔画之间 1 像素的空隙，使笔画密集的字仍然清晰。

## Platinum Mono

### 格子

每格宽 7 像素、高 14 像素：基线上 11 行，基线下 3 行，已含行间距。字符占的格数
与 `wcwidth` 一致：东亚宽字符和全角字符占两格，组合符号不占格，其余占一格，
东亚宽度不明确（Ambiguous）的字符也占一格。放不进格子的字形不收录，交给终端的
后备字体绘制。

每格的第一列留空，作为字符之间的间隔。只有延续到下一格的线条才会画到这一列：
破折号、制表符的线条和连字。

### 西文与符号

拉丁、希腊、西里尔字母和大多数符号取自缝合像素字体 12px 等宽版。原稿画在 6×12
的格子上，放进 Mono 7×14 的格子时右移 1 列，基线不变。

原稿画成两格宽或未收录的约一百个常用符号，如 — → ■ ✓ ≠，在
[`glyphs/mono.txt`](../glyphs/mono.txt) 中重新画成一格宽。

制表符、方块、阴影、盲文和 Powerline 分隔符由
[`internal/grid`](../internal/grid) 生成，填满格子并与相邻格子相接。阴影排列在
每 7 列重复一次的斜向网格上，使相邻的格子无缝衔接。盲文的每个点为 2×2 像素。

### 东亚文字

中文、日文、韩文使用对应地区的 `cjk12` 字形，右移 1 列。与其他字族一样，它们在
基线下占 1 行，因此同一行中的西文与东亚文字共用基线，并在上下各留出 1 行空白。

### Bold

Bold 在格子内由 Regular 加粗得到：每个字形向右偏移 1 像素再绘制一次，同时保留
笔画间的空隙和每格的第一列。程序生成的字形和图标在两种字重中相同。

### 连字

31 个连字，如 `=>` `!=` `<=` `->` `::` `/*`，取自
[AKR Pixel](https://github.com/AkaraChen/font)。原稿画在 6×12 的格子上，与西文
一样加宽。每个连字按格切成片，每个字形仍只占一格，文字依然对齐。

如果连字前一个字符与它的首字符相同，或后一个字符与它的末字符相同，就不连接，
因此 `====`、`....` 这样的连续字符保持分开。在字体文件中，连字以 OpenType 特性
`calt` 实现。

### 图标

图标为 [Nerd Fonts](https://www.nerdfonts.com) 3.5.1 私用区中的约 10,600 个。它们
由 Symbols Nerd Font 栅格化而来：字体的行高对应格子的 14 行，图标在两格中居中；
Powerline 符号保持在格子中的原位。

栅格化到这个尺寸，细节多的图标会失去形状。eza、Starship 和 Powerline 提示符最常
用的约 170 个图标在 [`glyphs/icons.txt`](../glyphs/icons.txt) 中重新绘制，包括
文件夹与文件、编程语言、操作系统、Git 和状态。它们画在 12×11 像素内，以汉字的
中线居中。

每个图标占一格、最宽画到两格，与 Nerd Font（而非 Nerd Font Mono）相同。
