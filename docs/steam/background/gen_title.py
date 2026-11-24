#!/usr/bin/env python3
# タイトル画面の背景を決定的に描く。凍った惑星のシーン planet_scene.html を、タイトル画面のピクセル絵の大きさ
# 320x240 で描いて色を減らし、title_master.png に書く。compose.sh がこれを3倍に広げてロゴを重ねる。
# 空の seed を変えると星雲と星の並びが変わる。構図を探すときは SKY を変えて見比べ、決めた seed を書く。
#
# 使い方: nix-shell ../shell.nix 内で python3 gen_title.py
# 依存: google-chrome

import os
import subprocess
import tempfile

from PIL import Image

import gen_crash

SKY = 16
W, H = 320, 240
COLORS = 32
SCENE = "file://" + os.path.join(gen_crash.HERE, "planet_scene.html")
OUT = os.path.join(gen_crash.HERE, "title_master.png")


def main():
    env = {k: v for k, v in os.environ.items() if k != "LD_LIBRARY_PATH"}
    with tempfile.TemporaryDirectory() as tmp:
        raw = os.path.join(tmp, "raw.png")
        subprocess.run(
            [
                "google-chrome", "--headless=new", "--enable-unsafe-swiftshader", "--use-angle=swiftshader",
                f"--window-size={W},{H + 200}", "--hide-scrollbars", "--virtual-time-budget=20000",
                f"--screenshot={raw}", f"{SCENE}?w={W}&h={H}&sky={SKY}",
            ],
            env=env, check=True, capture_output=True, text=True,
        )
        im = Image.open(raw).convert("RGB").crop((0, 0, W, H))
    if im.getcolors(COLORS) is not None:
        raise RuntimeError("描き損じた: 色が少なすぎる")
    im.quantize(colors=COLORS, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE).convert("RGB").save(OUT)
    print(f"saved {OUT}", flush=True)


if __name__ == "__main__":
    main()
