#!/usr/bin/env python3
# マスター画像を Stable Diffusion (SD Turbo) で生成する
#
# 依存:
#   pip install -r docs/steam/requirements.txt
#
# Nix の Python では libcuda.so.1 と libstdc++.so.6 が見えないため、
# シンボリックリンクを別ディレクトリに集めて LD_LIBRARY_PATH で指定する:
#   mkdir -p /tmp/nvidia-libs
#   ln -sf /usr/lib/x86_64-linux-gnu/libcuda.so.1 /tmp/nvidia-libs/
#   ln -sf /usr/lib/x86_64-linux-gnu/libnvidia-ml.so.1 /tmp/nvidia-libs/
#   ln -sf "$(find /nix/store -name 'libstdc++.so.6' -path '*/gcc-*-lib/*' | head -1)" /tmp/nvidia-libs/
#
# 使い方:
#   LD_LIBRARY_PATH=/tmp/nvidia-libs python docs/steam/gen_master.py

import torch
from diffusers import AutoPipelineForImage2Image
from PIL import Image

pipe = AutoPipelineForImage2Image.from_pretrained(
    "stabilityai/sd-turbo",
    torch_dtype=torch.float16,
    variant="fp16",
)
pipe.enable_model_cpu_offload()
pipe.enable_attention_slicing()
pipe.enable_vae_tiling()

def pixelate(image, scale):
    w, h = image.size
    small = image.resize((w // scale, h // scale), Image.LANCZOS)
    return small.resize((w, h), Image.NEAREST)

img = Image.open("docs/steam/source/dungeon.jpg").convert("RGB")

prompt = "dark ancient stone dungeon, tall stone archway in the center leading to another dark room beyond, endless corridors receding into darkness, massive bright blue glowing crystal formations covering the walls and ceiling, large ice crystal clusters everywhere, frozen crystalline surfaces, frost and ice on stone, warm orange torch light from above contrasting with blue crystals, symmetrical composition, cold dark atmosphere, fantasy dungeon, dramatic depth, detailed, high quality"
neg = "blurry, text, watermark, person, character, bright, overexposed, dead end, outdoor, empty walls, plain walls"

# dungeon.jpg (5767x3845 ≈ 1.5:1) を 960x640 にリサイズ
cropped = img.resize((960, 640), Image.LANCZOS)

gen = torch.Generator("cuda").manual_seed(23)
result = pipe(
    prompt=prompt, negative_prompt=neg, image=cropped,
    strength=0.75, guidance_scale=2.0, num_inference_steps=6,
    generator=gen,
).images[0]

# 高精細のまま LANCZOS で拡大して保存する
# ピクセルアート化と暗化は crop_assets.sh で各アセットサイズに応じて行う
master = result.resize((3840, 2560), Image.LANCZOS)

master.save("docs/steam/generated/master_3840x2560.png")
print(f"Master saved: {master.size}")

# 縦長カプセル用マスター (640x960)
# 元画像を縦長にクロップしてから生成する
vert_crop = img.crop((
    img.width // 2 - img.height * 2 // 3,  # 中央付近を縦長に切り出し
    0,
    img.width // 2 + img.height * 2 // 3,
    img.height,
)).resize((640, 960), Image.LANCZOS)

gen_v = torch.Generator("cuda").manual_seed(24)
result_v = pipe(
    prompt=prompt, negative_prompt=neg, image=vert_crop,
    strength=0.75, guidance_scale=2.0, num_inference_steps=6,
    generator=gen_v,
).images[0]

master_v = result_v.resize((2560, 3840), Image.LANCZOS)
master_v.save("docs/steam/generated/master_vert_2560x3840.png")
print(f"Master (vertical) saved: {master_v.size}")
