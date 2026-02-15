package screeneffect

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/assets"
)

// RetroFilter はレトロゲーム風のエフェクトを適用するフィルタ
type RetroFilter struct {
	shader    *ebiten.Shader
	startTime time.Time
	enabled   bool
}

// NewRetroFilter は新しいレトロフィルタを作成する
func NewRetroFilter() *RetroFilter {
	filter := &RetroFilter{
		enabled:   true,
		startTime: time.Now(),
	}
	filter.initShader()
	return filter
}

// initShader はシェーダーを初期化する
func (f *RetroFilter) initShader() {
	if f.shader != nil {
		return
	}

	shaderSrc, err := assets.FS.ReadFile("file/shaders/retro.kage")
	if err != nil {
		f.enabled = false
		return
	}

	shader, err := ebiten.NewShader(shaderSrc)
	if err != nil {
		f.enabled = false
		return
	}

	f.shader = shader
}

// Apply はFilterインターフェースの実装
// ソース画像にエフェクトを適用して描画先に出力する
func (f *RetroFilter) Apply(dst, src *ebiten.Image) {
	if !f.enabled || f.shader == nil {
		dst.DrawImage(src, nil)
		return
	}

	bounds := src.Bounds()
	width := float32(bounds.Dx())
	height := float32(bounds.Dy())
	elapsed := float32(time.Since(f.startTime).Seconds())

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]interface{}{
		"Time":       elapsed,
		"ScreenSize": []float32{width, height},
	}
	op.Images[0] = src

	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), f.shader, op)
}
