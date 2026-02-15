package screeneffect

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/assets"
)

// RetroFilter はレトロゲーム風のエフェクトを適用するフィルタ
type RetroFilter struct {
	shader *ebiten.Shader
}

// NewRetroFilter は新しいレトロフィルタを作成する
func NewRetroFilter() (*RetroFilter, error) {
	filter := &RetroFilter{}
	if err := filter.initShader(); err != nil {
		return nil, err
	}
	return filter, nil
}

// initShader はシェーダーを初期化する
func (f *RetroFilter) initShader() error {
	if f.shader != nil {
		return nil
	}

	shaderSrc, err := assets.FS.ReadFile("file/shaders/retro.kage")
	if err != nil {
		return fmt.Errorf("シェーダーファイルの読み込みに失敗: %w", err)
	}

	shader, err := ebiten.NewShader(shaderSrc)
	if err != nil {
		return fmt.Errorf("シェーダーのコンパイルに失敗: %w", err)
	}

	f.shader = shader
	return nil
}

// Apply はFilterインターフェースの実装
// ソース画像にエフェクトを適用して描画先に出力する
func (f *RetroFilter) Apply(dst, src *ebiten.Image) {
	if f.shader == nil {
		dst.DrawImage(src, nil)
		return
	}

	bounds := src.Bounds()
	width := float32(bounds.Dx())
	height := float32(bounds.Dy())

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]interface{}{
		"ScreenSize": []float32{width, height},
	}
	op.Images[0] = src

	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), f.shader, op)
}
