#!/usr/bin/env python3
# 墜落現場のシーン crash_scene.html を seed ごとに描き、ピクセル絵にして一覧に並べる。
# 小さく描いて色を減らし、最近傍で拡大する。炎の橙は面積が小さく自動の減色で消えるので固定で足す。
#
# 使い方: nix-shell ../shell.nix 内で
#   python3 gen_crash.py 7            seed 7 の基準の構図だけ描く
#   python3 gen_crash.py 100-147      範囲の seed を描き、12枚ずつ一覧にする
# 依存: google-chrome。出力は ./crash-out/ に置く。

import os
import subprocess
import sys
import tempfile

from PIL import Image, ImageDraw

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "crash-out")
SCENE = "file://" + os.path.join(HERE, "crash_scene.html")

W, H = 480, 320
SCALE = 8
AUTO_COLORS = 26
WARM = [
    0xFF, 0x6A, 0x1F, 0xFF, 0xB3, 0x47, 0xFF, 0xE0, 0x8A,
    0xC0, 0x44, 0x1A, 0x7A, 0x30, 0x1A, 0x3A, 0x1E, 0x18,
]


def render(seed, path):
    env = {k: v for k, v in os.environ.items() if k != "LD_LIBRARY_PATH"}
    subprocess.run(
        [
            "google-chrome", "--headless=new", "--enable-unsafe-swiftshader", "--use-angle=swiftshader",
            f"--window-size={W},{H + 200}", "--hide-scrollbars", "--virtual-time-budget=20000",
            f"--screenshot={path}", f"{SCENE}?w={W}&h={H}&px=1&seed={seed}",
        ],
        env=env, check=True, capture_output=True,
    )


def pixelize(raw):
    im = Image.open(raw).convert("RGB").crop((0, 0, W, H))
    auto = im.quantize(colors=AUTO_COLORS, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
    pal = Image.new("P", (1, 1))
    colors = auto.getpalette()[: AUTO_COLORS * 3] + WARM
    pal.putpalette(colors + [0] * (768 - len(colors)))
    return im.quantize(palette=pal, dither=Image.Dither.NONE).convert("RGB")


def seeds_of(args):
    for a in args:
        if "-" in a:
            lo, hi = map(int, a.split("-"))
            yield from range(lo, hi + 1)
        else:
            yield int(a)


def sheet(items, path):
    cols, tw, th = 4, W, H
    rows = (len(items) + cols - 1) // cols
    s = Image.new("RGB", (cols * tw, rows * th), "black")
    d = ImageDraw.Draw(s)
    for i, (seed, im) in enumerate(items):
        x, y = (i % cols) * tw, (i // cols) * th
        s.paste(im, (x, y))
        d.rectangle((x, y, x + 34, y + 12), fill="black")
        d.text((x + 3, y + 1), str(seed), fill="white")
    s.resize((s.width * 2, s.height * 2), Image.NEAREST).save(path)


def main():
    os.makedirs(OUT, exist_ok=True)
    done = []
    with tempfile.TemporaryDirectory() as tmp:
        for seed in seeds_of(sys.argv[1:] or ["7"]):
            raw = os.path.join(tmp, f"{seed}.png")
            render(seed, raw)
            px = pixelize(raw)
            px.resize((W * SCALE, H * SCALE), Image.NEAREST).save(os.path.join(OUT, f"crash_{seed:04d}.png"))
            done.append((seed, px))
            print(f"crash_{seed:04d}.png", flush=True)
    for i in range(0, len(done), 12):
        chunk = done[i : i + 12]
        sheet(chunk, os.path.join(OUT, f"sheet_{chunk[0][0]:04d}-{chunk[-1][0]:04d}.png"))


main()
