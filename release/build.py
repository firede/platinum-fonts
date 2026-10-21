"""Makes the release's font packages from the BDF fonts cmd/export writes.

Each BDF font becomes OTF, TTF, WOFF2, OTB, BDF and PCF; each format is
packed in a zip with the licenses. The OpenType feature file of the same
name next to a BDF font, if any, goes into the OpenType formats and OTB.

    python release/build.py --version 0.1.0 --date 2026-10-15 dist/bdf dist
"""

import argparse
import os
import re
import zipfile
from concurrent.futures import ProcessPoolExecutor
from datetime import datetime, timezone
from pathlib import Path

import bdffont
from pixel_font_builder import FontBuilder, Glyph, SerifStyle, SlantStyle, WeightName, WidthStyle, opentype

ROOT = Path(__file__).resolve().parent.parent
REPOSITORY = 'https://github.com/firede/platinum-fonts'
FORMATS = ['otf', 'ttf', 'woff2', 'otb', 'bdf', 'pcf']


def copyright_lines(path: Path) -> list[str]:
    """Returns the copyright lines of a license file, as far as its terms."""
    head = path.read_text().split('This Font Software is licensed')[0].split('Permission is hereby granted')[0]
    return [line.strip() for line in head.splitlines() if line.strip().lower().startswith('copyright')]


OFL = 'This Font Software is licensed under the SIL Open Font License, Version 1.1.'
# Platinum Mono's ligatures and icons come under licenses of their own.
MONO_LICENSES = [ROOT / 'licenses/akr-pixel/LICENSE.txt', ROOT / 'licenses/nerd-fonts/LICENSE.txt']
MONO_LICENSE = OFL + (' Its ligatures, from AKR Pixel, are under the MIT License; its icons, from'
                      ' Nerd Fonts, under the licenses of their icon sets (MIT, CC BY 4.0, Apache 2.0,'
                      ' OFL 1.1). See NOTICE.md.')


def make_builder(bdf_path: Path, version: str, date: datetime) -> FontBuilder:
    font = bdffont.BdfFont.load(bdf_path)
    props = font.properties
    size = int(props['PIXEL_SIZE'])
    ascent, descent = int(props['FONT_ASCENT']), int(props['FONT_DESCENT'])

    builder = FontBuilder()
    metric = builder.font_metric
    metric.font_size = size
    metric.horizontal_layout.ascent = ascent
    metric.horizontal_layout.descent = -descent
    metric.vertical_layout.ascent = size // 2
    metric.vertical_layout.descent = -size // 2
    metric.x_height = int(props['X_HEIGHT'])
    metric.cap_height = int(props['CAP_HEIGHT'])
    metric.underline_position = -2
    metric.underline_thickness = 1
    metric.strikeout_position = metric.x_height // 2
    metric.strikeout_thickness = 1

    family = props['FAMILY_NAME']
    bold = props['WEIGHT_NAME'] == 'Bold'
    mono = props.get('SPACING') == 'M'
    builder.opentype_config.is_fixed_pitch = mono
    meta = builder.meta_info
    meta.version = version
    meta.created_time = date
    meta.modified_time = date
    meta.family_name = family
    meta.weight_name = WeightName.BOLD if bold else WeightName.REGULAR
    meta.serif_style = SerifStyle.SANS_SERIF
    meta.slant_style = SlantStyle.NORMAL
    meta.width_style = WidthStyle.MONOSPACED if mono else WidthStyle.PROPORTIONAL
    meta.manufacturer = 'Platinum Fonts'
    if mono:
        cell_width = next(g.device_width_x for g in font.glyphs if g.encoding == 0x20)
        meta.designer = 'Firede; TakWolf and others (Latin and East Asian glyphs, from Fusion Pixel Font)'
        meta.description = f'{family}: a monospaced bitmap face for terminals and code, in cells {cell_width}px wide and {ascent + descent}px tall. Use it at multiples of {size}px.'
    else:
        meta.designer = 'Firede; TakWolf and others (East Asian glyphs, from Fusion Pixel Font)'
        meta.description = f'{family}: a bitmap face on a {size}px grid. Use it at multiples of {size}px.'
    holders = copyright_lines(ROOT / 'OFL.txt')
    if mono:
        holders += [line for path in MONO_LICENSES for line in copyright_lines(path)]
    meta.copyright_info = '\n'.join(holders)
    meta.license_info = MONO_LICENSE if mono else OFL
    meta.vendor_url = REPOSITORY
    meta.designer_url = REPOSITORY
    meta.license_url = 'https://openfontlicense.org'

    # Vertical text: each glyph centered on the vertical axis, in an em box
    # whose top is 10 pixels over the horizontal baseline.
    em_top = size - 2
    for g in font.glyphs:
        width, height = g.dimensions
        offset_x, offset_y = g.offset
        builder.glyphs.append(Glyph(
            name=g.name,
            horizontal_offset=(offset_x, offset_y),
            advance_width=g.device_width_x,
            vertical_offset=(offset_x - g.device_width_x // 2, em_top - (offset_y + height)),
            advance_height=size,
            bitmap=g.bitmap,
        ))
        if g.encoding >= 0:
            builder.character_mapping[g.encoding] = g.name
    # Ligatures: cmd/export writes the feature that draws them next to the
    # font.
    fea = bdf_path.with_suffix('.fea')
    if fea.exists():
        builder.opentype_config.features = opentype.FeatureFile(fea)
    return builder


def postscript_name(builder: FontBuilder) -> str:
    style = 'Bold' if builder.meta_info.weight_name == WeightName.BOLD else 'Regular'
    return builder.meta_info.family_name.replace(' ', '') + '-' + style


def finish(builder: FontBuilder, fb) -> None:
    """Names the font as PostScript names are written, and marks its style
    where Windows and macOS look for it, so that Regular and Bold make a
    family; and a monospaced font as monospaced, where Windows looks, its
    average width that of a cell, as East Asian glyphs taking two would
    otherwise make it wider."""
    ps = postscript_name(builder)
    names = fb.font['name']
    for record in list(names.names):
        if record.nameID == 6:
            names.setName(ps, 6, record.platformID, record.platEncID, record.langID)
        elif record.nameID == 3:
            names.setName(f'{ps};{builder.meta_info.version}', 3, record.platformID, record.platEncID, record.langID)
    bold = builder.meta_info.weight_name == WeightName.BOLD
    fb.font['OS/2'].fsSelection = 0x20 if bold else 0x40
    fb.font['head'].macStyle = 0x01 if bold else 0x00
    if builder.meta_info.width_style == WidthStyle.MONOSPACED:
        panose = fb.font['OS/2'].panose
        panose.bFamilyType, panose.bProportion = 2, 9  # Latin text, monospaced
        space = next(g for g in builder.glyphs if g.name == builder.character_mapping[0x20])
        units = fb.font['head'].unitsPerEm // builder.font_metric.font_size
        fb.font['OS/2'].xAvgCharWidth = space.advance_width * units


def save(builder: FontBuilder, fmt: str, path: Path) -> None:
    if fmt in ('otf', 'ttf', 'woff2', 'otb'):
        fb = {
            'otf': lambda: builder.to_otf_builder(),
            'ttf': lambda: builder.to_ttf_builder(),
            'woff2': lambda: builder.to_otf_builder(flavor=opentype.Flavor.WOFF2),
            'otb': lambda: builder.to_otb_builder(),
        }[fmt]()
        finish(builder, fb)
        fb.save(path)
    elif fmt == 'bdf':
        builder.save_bdf(path)
    else:
        builder.save_pcf(path)


def pack(out: Path, fmt: str, version: str, date: datetime, fonts: list[Path]) -> Path:
    """Zips the fonts of a format with the licenses, dated date."""
    name = f'platinum-fonts-v{version}-{fmt}'
    zip_path = out / f'{name}.zip'
    stamp = date.timetuple()[:6]
    extras = [ROOT / 'OFL.txt', ROOT / 'NOTICE.md', *sorted((ROOT / 'licenses').rglob('*.txt'))]
    with zipfile.ZipFile(zip_path, 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for path in sorted(fonts):
            z.writestr(zipfile.ZipInfo(f'{name}/{path.name}', stamp), path.read_bytes(), zipfile.ZIP_DEFLATED)
        for path in extras:
            z.writestr(zipfile.ZipInfo(f'{name}/{path.relative_to(ROOT).as_posix()}', stamp), path.read_bytes(), zipfile.ZIP_DEFLATED)
    return zip_path


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument('--version', required=True, help='the version, as 0.1.0')
    parser.add_argument('--date', required=True, help='the date of the release, as 2026-10-15')
    parser.add_argument('--formats', default=','.join(FORMATS), help='the formats to make')
    parser.add_argument('--jobs', type=int, default=os.cpu_count(), help='the fonts to make at once')
    parser.add_argument('bdf_dir', type=Path)
    parser.add_argument('out_dir', type=Path)
    args = parser.parse_args()
    if not re.fullmatch(r'\d+\.\d+\.\d+', args.version):
        parser.error('--version must be as 0.1.0')
    date = datetime.fromisoformat(args.date).replace(tzinfo=timezone.utc)
    formats = args.formats.split(',')

    bdfs = sorted(args.bdf_dir.glob('*.bdf'))
    if not bdfs:
        parser.error(f'no BDF fonts in {args.bdf_dir}')
    made = {fmt: [] for fmt in formats}
    with ProcessPoolExecutor(max_workers=args.jobs) as pool:
        jobs = [pool.submit(build_one, path, args.version, date, formats, args.out_dir) for path in bdfs]
        for job in jobs:
            for fmt, path in job.result().items():
                made[fmt].append(path)
                print('made', path)
    for fmt in formats:
        print('packed', pack(args.out_dir, fmt, args.version, date, made[fmt]))


def build_one(bdf_path: Path, version: str, date: datetime, formats: list[str], out_dir: Path) -> dict[str, Path]:
    """Makes a font in each of formats, in out_dir/FORMAT."""
    builder = make_builder(bdf_path, version, date)
    made = {}
    for fmt in formats:
        dir_ = out_dir / fmt
        dir_.mkdir(parents=True, exist_ok=True)
        path = dir_ / f'{postscript_name(builder)}.{fmt}'
        save(builder, fmt, path)
        made[fmt] = path
    return made


if __name__ == '__main__':
    main()
