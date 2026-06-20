#!/usr/bin/env python3
import torch
from diffusers import AutoPipelineForImage2Image
from PIL import Image, ImageEnhance

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

def darken(image, factor):
    return ImageEnhance.Brightness(image).enhance(factor)

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
