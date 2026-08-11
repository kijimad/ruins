package screeneffect

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/assets"
)

// LightFilter はワールドレイヤへ雰囲気ライティングをかけるフィルタ。
// 配置された光源から作ったライトマップを世界へ乗算し、仕上げにビネットとコントラストをかける。
// 光だまりの位置や色はライトマップが持つので、実際の LightSource がそのまま雰囲気になる。
type LightFilter struct {
	shader *ebiten.Shader
	// Darkness は暗さの度合い。0 で原画のまま、1 で最も暗い。Apply の前に呼び出し側が設定する。
	Darkness float32
}

// NewLightFilter は新しいライティングフィルタを作る。
func NewLightFilter() (*LightFilter, error) {
	shaderSrc, err := assets.FS.ReadFile("file/shaders/light.kage")
	if err != nil {
		return nil, fmt.Errorf("failed to load light shader: %w", err)
	}
	shader, err := ebiten.NewShader(shaderSrc)
	if err != nil {
		return nil, fmt.Errorf("failed to compile light shader: %w", err)
	}
	return &LightFilter{shader: shader}, nil
}

// Apply は仕上げのビネットとコントラストを src へかけて dst へ出力する。
// ライトマップの乗算は呼び出し側が済ませた前提で、ここは全体の締めだけ担う。
func (f *LightFilter) Apply(dst, src *ebiten.Image) {
	if f == nil || dst == nil || src == nil {
		return
	}
	if f.shader == nil {
		dst.DrawImage(src, nil)
		return
	}

	b := src.Bounds()
	w := float32(b.Dx())
	h := float32(b.Dy())

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"ScreenSize": []float32{w, h},
		"Darkness":   f.Darkness,
	}
	op.Images[0] = src
	dst.DrawRectShader(b.Dx(), b.Dy(), f.shader, op)
}
