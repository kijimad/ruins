package systems

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/assets"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/widgets/render"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

var whiteSilhouetteShader *ebiten.Shader

// VisualEffectSystem はビジュアルエフェクトを管理するシステム
type VisualEffectSystem struct{}

// String はシステム名を返す
func (sys VisualEffectSystem) String() string {
	return "VisualEffectSystem"
}

// Update はエフェクトを更新する
func (sys *VisualEffectSystem) Update(world w.World) error {
	var entitiesToDelete []ecs.Entity

	world.Manager.Join(
		world.Components.VisualEffect,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entitiesToDelete = append(entitiesToDelete, entity)
	}))

	// アニメーション無効時は即座に削除
	if world.Config.DisableAnimation {
		for _, entity := range entitiesToDelete {
			world.Manager.DeleteEntity(entity)
		}
		return nil
	}

	// アニメーション有効時は通常の更新処理
	const deltaMs = 1000.0 / 60.0 // 1フレームあたりの時間（60FPS想定）
	entitiesToDelete = entitiesToDelete[:0]

	world.Manager.Join(
		world.Components.VisualEffect,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		ve := world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)

		// エフェクトを更新
		activeEffects := ve.Effects[:0]
		for _, effect := range ve.Effects {
			// まだ継続中のエフェクトは保持する
			if effect.Update(deltaMs) {
				activeEffects = append(activeEffects, effect)
			}
		}
		ve.Effects = activeEffects

		// エフェクトがなくなったらエンティティを削除する
		if len(ve.Effects) == 0 {
			entitiesToDelete = append(entitiesToDelete, entity)
		}
	}))

	// エフェクト専用エンティティを削除
	for _, entity := range entitiesToDelete {
		world.Manager.DeleteEntity(entity)
	}

	return nil
}

// Draw はエフェクトを描画する
func (sys *VisualEffectSystem) Draw(world w.World, screen *ebiten.Image) error {
	if world.Resources.UIResources == nil || world.Resources.UIResources.Text == nil {
		return nil
	}
	face := world.Resources.UIResources.Text.TitleFontFace
	smallFace := world.Resources.UIResources.Text.SmallFace
	if face == nil || smallFace == nil {
		return nil
	}

	var err error
	world.Manager.Join(
		world.Components.VisualEffect,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		ve := world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)

		for _, effect := range ve.Effects {
			switch e := effect.(type) {
			case *gc.ScreenTextEffect:
				// 画面座標で描画（ダンジョンタイトルなど）
				sys.drawScreenText(screen, face, e)
			case *gc.DamageTextEffect:
				// エンティティ座標で描画
				if entity.HasComponent(world.Components.GridElement) {
					gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
					sys.drawDamageText(world, screen, smallFace, gridElement, e)
				}
			case *gc.SpriteFadeoutEffect:
				// スプライトフェードアウトエフェクトを描画
				if entity.HasComponent(world.Components.GridElement) {
					gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
					err = sys.drawSpriteFadeoutEffect(world, screen, gridElement, e)
					if err != nil {
						return
					}
				}
			}
		}
	}))
	if err != nil {
		return err
	}

	return nil
}

// drawScreenText は画面座標でテキストを描画する
func (sys *VisualEffectSystem) drawScreenText(screen *ebiten.Image, face text.Face, effect *gc.ScreenTextEffect) {
	if effect.Alpha <= 0 {
		return
	}

	// テキストサイズを測定して中央揃え
	textWidth, textHeight := text.Measure(effect.Text, face, 0)
	x := effect.OffsetX - textWidth/2
	y := effect.OffsetY - textHeight/2

	// 透明度を適用した色
	alpha := uint8(effect.Alpha * 255)
	textColor := color.RGBA{effect.Color.R, effect.Color.G, effect.Color.B, alpha}
	outlineColor := color.RGBA{0, 0, 0, alpha}

	// アウトライン付きテキストを描画
	render.OutlinedText(screen, effect.Text, face, x, y, textColor, outlineColor)
}

// drawDamageText はエンティティ座標でダメージテキストを描画する
func (sys *VisualEffectSystem) drawDamageText(world w.World, screen *ebiten.Image, face text.Face, gridElement *gc.GridElement, effect *gc.DamageTextEffect) {
	// グリッド座標をピクセル座標に変換
	pixelX := float64(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2)
	pixelY := float64(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)

	// オフセットを適用
	pixelX += effect.OffsetX
	pixelY += effect.OffsetY

	// カメラ変換を適用して画面座標に変換
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(pixelX, pixelY)
	SetTranslate(world, op)
	screenX, screenY := op.GeoM.Apply(0, 0)

	// テキストサイズを測定して中央揃え
	textWidth, _ := text.Measure(effect.Text, face, 0)
	x := screenX - textWidth/2
	y := screenY

	// 透明度を適用した色
	alpha := uint8(effect.Alpha * 255)
	textColor := color.RGBA{effect.Color.R, effect.Color.G, effect.Color.B, alpha}
	outlineColor := color.RGBA{0, 0, 0, alpha}

	// アウトライン付きテキストを描画
	render.OutlinedText(screen, effect.Text, face, x, y, textColor, outlineColor)
}

// drawSpriteFadeoutEffect はスプライトの白シルエットフェードアウトエフェクトを描画する
func (sys *VisualEffectSystem) drawSpriteFadeoutEffect(world w.World, screen *ebiten.Image, gridElement *gc.GridElement, effect *gc.SpriteFadeoutEffect) error {
	if effect.Alpha <= 0 {
		return nil
	}
	if world.Resources.SpriteSheets == nil {
		return nil
	}

	// シェーダーを初期化（初回のみ）
	if whiteSilhouetteShader == nil {
		shaderSource, err := assets.FS.ReadFile("file/shaders/white_silhouette.kage")
		if err != nil {
			return err
		}
		whiteSilhouetteShader, err = ebiten.NewShader(shaderSource)
		if err != nil {
			return err
		}
	}

	// スプライトシートを取得
	spriteSheet, exists := (*world.Resources.SpriteSheets)[effect.SpriteSheetName]
	if !exists {
		return nil
	}

	// スプライトを取得
	sprite, exists := spriteSheet.Sprites[effect.SpriteKey]
	if !exists {
		return nil
	}

	// スプライト画像を切り出す
	texture := spriteSheet.Texture
	textureWidth := texture.Image.Bounds().Dx()
	textureHeight := texture.Image.Bounds().Dy()
	left := max(0, sprite.X)
	right := min(textureWidth, sprite.X+sprite.Width)
	top := max(0, sprite.Y)
	bottom := min(textureHeight, sprite.Y+sprite.Height)
	img := texture.Image.SubImage(image.Rect(left, top, right, bottom)).(*ebiten.Image)

	// グリッド座標をピクセル座標に変換
	pixelX := float64(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2)
	pixelY := float64(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)

	// シェーダー描画オプションを設定
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(float64(-sprite.Width/2), float64(-sprite.Height/2))
	op.GeoM.Translate(pixelX, pixelY)

	// カメラ変換を適用
	imgOp := &ebiten.DrawImageOptions{}
	imgOp.GeoM = op.GeoM
	SetTranslate(world, imgOp)
	op.GeoM = imgOp.GeoM

	// ソース画像を設定
	op.Images[0] = img

	// 透明度をシェーダーに渡す（ColorScaleのAlphaを使用）
	op.ColorScale.ScaleAlpha(float32(effect.Alpha))

	// シェーダーで白シルエットを描画
	screen.DrawRectShader(sprite.Width, sprite.Height, whiteSilhouetteShader, op)
	return nil
}
