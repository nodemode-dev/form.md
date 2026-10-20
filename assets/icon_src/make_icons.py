"""Build every form.md icon file from art1024.png.

art1024.png is opaque: the owner's tile on the light page it was mocked up
on, including a soft grey shadow at the lower right. The tile is cut out
with a superellipse (|x/a|^n + |y/a|^n = 1) fitted to stay 2 px inside the
dark tile on every side, drawn at 4x and averaged down, so no pixel of the
light page reaches any icon. Pixels outside the tile carry the tile's edge
colour at zero alpha, so a resampler that ignores alpha cannot pull in a
light fringe either.

Run from the repository root: python3 assets/icon_src/make_icons.py
Needs Pillow and NumPy.
"""

import base64
import io
import struct
from pathlib import Path

import numpy as np
from PIL import Image, ImageFilter

SRC = Path("assets/icon_src/art1024.png")
N = 5.40  # superellipse exponent, fitted to the tile's outline
A = 506.9  # half-width in px at 1024, 2 px inside the tile's dark edge
SS = 4  # supersampling for the edge


def tile_mask(size: int = 1024) -> Image.Image:
    big = size * SS
    c = (big - 1) / 2
    a = A * SS * size / 1024
    y, x = np.mgrid[0:big, 0:big].astype(np.float64)
    inside = (np.abs((x - c) / a) ** N + np.abs((y - c) / a) ** N) <= 1.0
    m = inside.reshape(size, SS, size, SS).mean(axis=(1, 3))
    return Image.fromarray(np.round(m * 255).astype(np.uint8), "L")


def cut_tile() -> Image.Image:
    art = Image.open(SRC).convert("RGB")
    mask = tile_mask(art.width)
    rgb = np.array(art).astype(np.float64)
    m = np.array(mask) > 0
    # Edge colour: mean of the opaque pixels in the outer 6 px of the tile.
    inner = np.array(mask.filter(ImageFilter.MinFilter(13))) == 255
    ring = m & ~inner
    edge = rgb[ring].mean(axis=0)
    rgb[~m] = edge
    out = np.dstack([rgb.round().astype(np.uint8), np.array(mask)])
    return Image.fromarray(out, "RGBA")


def resize(im: Image.Image, size: int) -> Image.Image:
    """Downscale with premultiplied alpha so edges stay dark."""
    return im.convert("RGBa").resize((size, size), Image.LANCZOS).convert("RGBA")


def macos(tile: Image.Image) -> Image.Image:
    """824 px tile on a 1024 canvas, 100 px margin, the family drop shadow."""
    t = resize(tile, 824)
    canvas = Image.new("RGBA", (1024, 1024), (0, 0, 0, 0))
    alpha = Image.new("L", (1024, 1024), 0)
    alpha.paste(t.getchannel("A"), (100, 110))
    shadow = alpha.filter(ImageFilter.GaussianBlur(14)).point(lambda v: int(v * 0.42))
    canvas.putalpha(shadow)
    canvas.alpha_composite(t, (100, 100))
    return canvas


def icns(im: Image.Image) -> bytes:
    kinds = [("icp4", 16), ("icp5", 32), ("icp6", 64), ("ic07", 128), ("ic08", 256), ("ic09", 512), ("ic10", 1024), ("ic11", 32), ("ic12", 64), ("ic13", 256), ("ic14", 512)]
    body = b""
    for kind, size in kinds:
        buf = io.BytesIO()
        resize(im, size).save(buf, "PNG", optimize=True)
        data = buf.getvalue()
        body += kind.encode("ascii") + struct.pack(">I", len(data) + 8) + data
    return b"icns" + struct.pack(">I", len(body) + 8) + body


def svg(im: Image.Image, px: int, shown: int) -> str:
    buf = io.BytesIO()
    resize(im, px).save(buf, "PNG", optimize=True)
    b64 = base64.b64encode(buf.getvalue()).decode("ascii")
    return f'<svg xmlns="http://www.w3.org/2000/svg" width="{shown}" height="{shown}" viewBox="0 0 {px} {px}"><image href="data:image/png;base64,{b64}" width="{px}" height="{px}"/></svg>'


def main() -> None:
    tile = cut_tile()
    mac = macos(tile)

    def png(im: Image.Image, path: str, size: int) -> None:
        resize(im, size).save(path, "PNG", optimize=True)

    tile.save("assets/form-md_icon.png", "PNG", optimize=True)
    tile.save("src-tauri/icons/icon.png", "PNG", optimize=True)
    mac.save("assets/form-md_icon_macos.png", "PNG", optimize=True)
    png(tile, "src-tauri/icons/32x32.png", 32)
    png(tile, "src-tauri/icons/128x128.png", 128)
    png(tile, "src-tauri/icons/128x128@2x.png", 256)
    png(tile, "src-tauri/splash/icon.png", 256)
    resize(tile, 256).save("src-tauri/icons/icon.ico", sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
    Path("src-tauri/icons/icon.icns").write_bytes(icns(mac))
    Path("web/public/favicon.svg").write_text(svg(tile, 192, 192))
    Path("assets/form-md_logo.svg").write_text(svg(tile, 320, 160))


if __name__ == "__main__":
    main()
