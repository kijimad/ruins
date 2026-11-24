#!/usr/bin/env python3
# 背景マスターを決定的に描く。墜落現場のシーン crash_scene.html の seed 203:9 を、
# gen_crash.py と同じ手順でピクセル絵にして 3840x2560 で書く。
# ストアの絵は墜落の瞬間の master、タイトル画面は同じ構図で炎が消えて時間がたった後の title_master にする。
# 構図を探すときは gen_crash.py で seed を並べて見比べ、決めた seed を KEY に書く。
#
# 使い方: nix-shell ../shell.nix 内で python3 gen_master.py
# 依存: google-chrome

import os
import tempfile

from PIL import Image

import gen_crash

KEY = "203:9"
OUT = os.path.join(gen_crash.HERE, "master_3840x2560.png")
TITLE_OUT = os.path.join(gen_crash.HERE, "title_master_3840x2560.png")


def draw(path, after):
    with tempfile.TemporaryDirectory() as tmp:
        raw = os.path.join(tmp, "raw.png")
        gen_crash.render(KEY, raw, after=after)
        px = gen_crash.pixelize(raw)
    master = px.resize((gen_crash.W * gen_crash.SCALE, gen_crash.H * gen_crash.SCALE), Image.NEAREST)
    master.save(path)
    print(f"saved {path} {master.size}", flush=True)


def main():
    draw(OUT, after=False)
    draw(TITLE_OUT, after=True)


if __name__ == "__main__":
    main()
