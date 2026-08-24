#!/usr/bin/env python3
# 背景マスター画像を決定的に生成する。横1枚だけ作る。
# 縦カプセルは compose がこの横マスターをポートレートに切り出して作るので、縦は生成しない。
#
# 絵柄は、深い青のマットな立方体が朝焼けの雪原に佇み、地平から昇る陽で手前へ1本の長い影が
# 伸びる2Dアニメ調のマット絵。参照はエヴァのラミエル。青は主役、面は不透明マットで直線の角。
#
# 生成は3段の連鎖。全段シード固定なので、モデルとパラメータが同じなら毎回同じ絵が出る。
#   1. SDXL txt2img でベースを描く          seed 91
#      影が1本になるシードを選んである。2股・分裂をネガティブで弾く
#   2. Ghibli img2img で2Dアニメ塗りへ変換   seed 128
#      img2img なのでキューブが落ちない。低strengthで足元の綿雪化を防ぐ
#   3. 色補正で雪を白く寄せ寒色に整える
# 最後に LANCZOS でマスター寸法へ伸ばす。
#
# ============================================================================
# 背景生成の環境構築と実行手順
# ============================================================================
# GPU は RTX 3050 6GB を前提とする。torch は CUDA 12.1 版を pip の wheel で入れる。
# nix は Python インタプリタの供給だけに使い、torch 本体は nix パッケージにしない。
# このユーザーは nix の trusted-user でないため cachix を引けず、cudaSupport 付き
# torch を nix でビルドすると全てソースビルドになり非現実的なため。
#
# 重要な前提。venv とモデルキャッシュはスクラッチパッドや /tmp に置かない。
# それらはセッションごとに消える。モデルは数GBあり再取得が重い。
# よってモデルキャッシュは HF_HOME=~/.cache/huggingface に固定し、venv も
# 消えない作業ディレクトリに作る。
#
# --- 1. torch cu121 の wheel が要求する Python 3.12 を nix で用意する ---
#   P312=$(nix build --no-link --print-out-paths nixpkgs#python312)/bin/python3.12
#
# --- 2. venv を作り依存を入れる。cu121 の index は requirements.txt 冒頭に書いてある ---
#   "$P312" -m venv sdvenv
#   ./sdvenv/bin/python -m pip install --upgrade pip
#   ./sdvenv/bin/python -m pip install \
#     --extra-index-url https://pypi.org/simple \
#     -r docs/steam/background/requirements.txt
#
# --- 3. nix の Python から見えない共有ライブラリを集めて LD_LIBRARY_PATH で渡す ---
# libcuda.so.1 と libnvidia-ml.so.1 はシステム側、libstdc++.so.6 は nix の gcc から取る。
#   mkdir -p /tmp/nvidia-libs
#   ln -sf /usr/lib/x86_64-linux-gnu/libcuda.so.1 /tmp/nvidia-libs/
#   ln -sf /usr/lib/x86_64-linux-gnu/libnvidia-ml.so.1 /tmp/nvidia-libs/
#   ln -sf "$(find /nix/store -name 'libstdc++.so.6' | grep -E 'gcc.*lib' | head -1)" /tmp/nvidia-libs/
#
# --- 4. 実行する。LD_LIBRARY_PATH と HF_HOME を必ず渡す ---
#   LD_LIBRARY_PATH=/tmp/nvidia-libs HF_HOME="$HOME/.cache/huggingface" \
#     ./sdvenv/bin/python docs/steam/background/gen_master.py
#
# 動作確認は次の一行で torch と CUDA を見る。cuda True が出れば準備完了。
#   LD_LIBRARY_PATH=/tmp/nvidia-libs ./sdvenv/bin/python -c \
#     "import torch; print(torch.__version__, torch.cuda.is_available())"
# ============================================================================

import gc
import os

import torch
from diffusers import AutoPipelineForImage2Image, StableDiffusionXLPipeline
from PIL import Image, ImageEnhance

os.chdir(os.path.dirname(os.path.abspath(__file__)))

SDXL = "stabilityai/stable-diffusion-xl-base-1.0"
GHIBLI = "nitrosocke/Ghibli-Diffusion"
OUT = "master_3840x2560.png"

# --- SDXL 段: 影が1本の深青キューブを平らな雪原に描く。2股をネガティブで弾く ---
SDXL_PROMPT = (
    "a large deep blue cube sitting on flat open snow on the left, "
    "low sun near the horizon, the cube backlit, "
    "one single long solid cast shadow stretching to the side across the flat snow, "
    "a clean unbroken shadow, smooth flat cobalt blue faces, sharp straight right-angle edges, "
    "three dimensional solid block, vast flat snowfield, distant snowy mountains at the horizon, "
    "glowing pink and orange dawn sky, matte painting, concept art, atmospheric, "
    "cinematic dawn light, minimalist composition, highly detailed"
)
SDXL_NEG = (
    "double shadow, two shadows, forked shadow, split shadow, divided shadow, streaky shadow, "
    "shadow with a gap, broken shadow, multiple shadows, "
    "buried cube, snow piled around the cube, fluffy snow mounds, "
    "irregular crystal, tilted cube, rotated, floating, levitating, "
    "mirror, chrome, glass box, landscape reflected on the cube, sun flare, lens flare, "
    "anime, cel shaded, cartoon, rounded, curved, sphere, "
    "person, text, watermark, logo, forest, trees, lake, water, ocean, reflection, "
    "blurry, low quality, distorted, deformed geometry, jpeg artifacts"
)

# --- Ghibli 段: 2Dアニメ塗りへ。足元の綿雪化を避け、白い雪と手前への影を促す ---
ANIME_PROMPT = (
    "ghibli style, a deep blue cube sitting on flat firm snow at dawn, "
    "a long soft cast shadow stretching across the flat snow in front of the cube, "
    "clean smooth snow ground at the base, bright white snow, cool blue shadows, "
    "distant snowy mountains, soft pale dawn sky, "
    "flat 2d anime background, hand painted, cel shading"
)
ANIME_NEG = (
    "fluffy snow mounds around the cube, cotton snow, snow piled high around the base, "
    "lumpy snow, snowdrift around the cube, snow nest, buried cube, "
    "sea of clouds, above the clouds, cloudscape, water, reflection, "
    "oversaturated, red snow, pink snow, magenta, 3d render, photo, glossy, "
    "blurry, text, watermark, people, sphere, rounded cube"
)


def free(pipe):
    del pipe
    gc.collect()
    torch.cuda.empty_cache()


# --- 1段目: SDXL txt2img でベースを描く ---
txt = StableDiffusionXLPipeline.from_pretrained(
    SDXL, torch_dtype=torch.float16, variant="fp16", use_safetensors=True
)
txt.enable_model_cpu_offload()
txt.enable_vae_tiling()
txt.enable_attention_slicing()
g = torch.Generator(device="cpu").manual_seed(91)
base = txt(
    prompt=SDXL_PROMPT, negative_prompt=SDXL_NEG, width=1152, height=768,
    guidance_scale=7.5, num_inference_steps=32, generator=g,
).images[0]
free(txt)

# --- 2段目: Ghibli img2img で2Dアニメ塗りへ。SD1.5 なので小さめ 832x554 で回す ---
ghibli = AutoPipelineForImage2Image.from_pretrained(
    GHIBLI, torch_dtype=torch.float16, safety_checker=None
)
ghibli.enable_model_cpu_offload()
ghibli.enable_attention_slicing()
g = torch.Generator(device="cpu").manual_seed(128)
anime = ghibli(
    prompt=ANIME_PROMPT, negative_prompt=ANIME_NEG,
    image=base.resize((832, 554), Image.LANCZOS),
    strength=0.34, guidance_scale=8.5, num_inference_steps=32, generator=g,
).images[0]
free(ghibli)

# --- 3段目: 色補正で雪を白く寄せ寒色に整える ---
# ImageMagick の -modulate 108,55,100 と -fill '#dfeaff' -colorize 7% を PIL で等価に置く。
graded = ImageEnhance.Brightness(anime).enhance(1.08)
graded = ImageEnhance.Color(graded).enhance(0.55)
tint = Image.new("RGB", graded.size, (0xDF, 0xEA, 0xFF))
graded = Image.blend(graded, tint, 0.07)

master = graded.resize((3840, 2560), Image.LANCZOS)
master.save(OUT)
print(f"saved {OUT} {master.size}", flush=True)
print("done")
