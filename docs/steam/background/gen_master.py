#!/usr/bin/env python3
# 背景マスターを決定的に描く。墜落現場のシーン crash_scene.html の seed 203:9 を、
# gen_crash.py と同じ手順でピクセル絵にして 3840x2560 で書く。
# ストアの絵は墜落の瞬間の master にする。タイトル画面の title_master は同じ構図で炎が消えて時間がたった後の夜にし、
# 雪原に浮かぶ動く家のキューブと焚き火を置く。冷たい世界の中でそこだけ暖かい、という暮らしの約束を見せる。
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
# キャンプはカメラから前へ 28、横へ 0 の所に置く
TITLE_QUERY = "&after=1&camp=28,0"


def draw(path, query):
    with tempfile.TemporaryDirectory() as tmp:
        raw = os.path.join(tmp, "raw.png")
        gen_crash.render(KEY, raw, query)
        px = gen_crash.pixelize(raw)
    master = px.resize((gen_crash.W * gen_crash.SCALE, gen_crash.H * gen_crash.SCALE), Image.NEAREST)
    master.save(path)
    print(f"saved {path} {master.size}", flush=True)


def main():
    draw(OUT, "")
    draw(TITLE_OUT, TITLE_QUERY)


if __name__ == "__main__":
    main()
