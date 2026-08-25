#!/usr/bin/env python3
# 背景マスターを決定的に生成する。横1枚だけ作り、縦カプセルは compose が横から切り出す。
# 全段シード固定で再現可能。
#   1. SDXL txt2img でベース          seed 91
#   2. Ghibli img2img で2Dアニメ塗り   seed 128
#   3. PIL で色補正し 3840x2560 へ拡大
#
# 環境: RTX 3050 6GB。torch は cu121 の wheel。python3.12 は docs/steam/shell.nix が供給する。
# venv とモデルキャッシュはこのディレクトリに作る。片付けは rm -rf sdvenv hf-cache。
#   cd docs/steam/background
#   nix-shell ../shell.nix
#   python3.12 -m venv sdvenv
#   ./sdvenv/bin/python -m pip install --extra-index-url https://pypi.org/simple -r pip-deps.txt
#   mkdir -p /tmp/nvidia-libs
#   ln -sf /usr/lib/x86_64-linux-gnu/libcuda.so.1 /usr/lib/x86_64-linux-gnu/libnvidia-ml.so.1 /tmp/nvidia-libs/
#   ln -sf "$(find /nix/store -name libstdc++.so.6 | grep -E 'gcc.*lib' | head -1)" /tmp/nvidia-libs/
#   LD_LIBRARY_PATH=/tmp/nvidia-libs HF_HOME=./hf-cache ./sdvenv/bin/python gen_master.py

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

# --- 2段目: Ghibli img2img で2Dアニメ塗りへ。1152x768 で回して解像感を稼ぐ ---
ghibli = AutoPipelineForImage2Image.from_pretrained(
    GHIBLI, torch_dtype=torch.float16, safety_checker=None
)
ghibli.enable_model_cpu_offload()
ghibli.enable_attention_slicing()
g = torch.Generator(device="cpu").manual_seed(128)
anime = ghibli(
    prompt=ANIME_PROMPT, negative_prompt=ANIME_NEG,
    image=base.resize((1152, 768), Image.LANCZOS),
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
