package systems

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/assets"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// VisualEffectSystem はビジュアルエフェクトを管理するシステム
type VisualEffectSystem struct {
	silhouetteShader *ebiten.Shader
}

// String はシステム名を返す
func (sys VisualEffectSystem) String() string {
	return "VisualEffectSystem"
}

// Update はエフェクトを更新する
func (sys *VisualEffectSystem) Update(world w.World) error {
	var entitiesToDelete []ecs.Entity

	collectQuery := query.ActiveFilter1[gc.VisualEffects](world).Query()
	for collectQuery.Next() {
		entity := collectQuery.Entity()
		entitiesToDelete = append(entitiesToDelete, entity)
	}

	// アニメーション無効時は即座に削除
	if world.Config.DisableAnimation {
		for _, entity := range entitiesToDelete {
			world.ECS.RemoveEntity(entity)
		}
		return nil
	}

	// アニメーション有効時は通常の更新処理
	const deltaMs = 1000.0 / 60.0 // 1フレームあたりの時間（60FPS想定）
	entitiesToDelete = entitiesToDelete[:0]

	updateQuery := query.ActiveFilter1[gc.VisualEffects](world).Query()
	for updateQuery.Next() {
		entity := updateQuery.Entity()
		ve := world.Components.VisualEffects.Get(entity)

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
	}

	// エフェクト専用エンティティを削除
	for _, entity := range entitiesToDelete {
		world.ECS.RemoveEntity(entity)
	}

	return nil
}

// Draw はエフェクトを描画する
func (sys *VisualEffectSystem) Draw(world w.World, screen *ebiten.Image) error {
	if world.Resources.UIResources.Text == nil {
		return nil
	}
	face := world.Resources.UIResources.Text.TitleFontFace
	smallFace := world.Resources.UIResources.Text.SmallFace
	if face == nil || smallFace == nil {
		return nil
	}

	// 投影はフレーム内で不変。ここで1回だけ組み、エンティティごとの描画へ渡す
	projector := render3d.ProjectorFor(world)

	var err error
	drawQuery := query.ActiveFilter1[gc.VisualEffects](world).Query()
	for drawQuery.Next() {
		entity := drawQuery.Entity()
		if err != nil {
			continue
		}
		ve := world.Components.VisualEffects.Get(entity)

	effectLoop:
		for _, effect := range ve.Effects {
			switch e := effect.(type) {
			case *gc.SplashTextEffect:
				sys.drawSplashText(world, screen, e)
			case *gc.DamageTextEffect:
				if world.Components.GridElement.Has(entity) {
					gridElement := world.Components.GridElement.Get(entity)
					sys.drawDamageText(screen, projector, smallFace, gridElement, e)
				}
			case *gc.SpriteFadeoutEffect:
				if world.Components.GridElement.Has(entity) {
					gridElement := world.Components.GridElement.Get(entity)
					err = sys.drawSpriteFadeoutEffect(world, screen, projector, gridElement, e)
					if err != nil {
						break effectLoop
					}
				}
			}
		}
	}

	return err
}

// drawSplashText はスプラッシュテキストを画面座標で描画する。
// テキストとラインをオフスクリーンバッファにフル不透明で描画し、
// バッファごとアルファを適用することでフェードタイミングを一致させる
func (sys *VisualEffectSystem) drawSplashText(world w.World, screen *ebiten.Image, effect *gc.SplashTextEffect) {
	if effect.Alpha <= 0 {
		return
	}

	screenW, screenH := screen.Bounds().Dx(), screen.Bounds().Dy()
	buf := ebiten.NewImage(screenW, screenH)

	// テキストサイズを測定して中央揃え
	textWidth, textHeight := text.Measure(effect.Text, effect.Face, 0)
	x := effect.Offset.X - textWidth/2
	y := effect.Offset.Y - textHeight/2

	// フル不透明でバッファに描画する
	textColor := effect.Color
	outlineColor := color.RGBA{0, 0, 0, 255}

	// グラデーション影を描画する。遠いレイヤーから順に描画して近いレイヤーで上書きする
	shadowLayers := [...]struct {
		offset float64
		alpha  uint8
	}{
		{5, 30},
		{4, 50},
		{3, 80},
		{2, 120},
		{1, 160},
	}
	shadowOp := &text.DrawOptions{}
	for _, layer := range shadowLayers {
		shadowOp.GeoM.Reset()
		shadowOp.GeoM.Translate(x+layer.offset, y+layer.offset)
		shadowOp.ColorScale.Reset()
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, layer.alpha})
		text.Draw(buf, effect.Text, effect.Face, shadowOp)
	}

	hud.OutlinedText(buf, effect.Text, effect.Face, x, y, textColor, outlineColor)

	if effect.LineWidth > 0 {
		lineY := y + textHeight + 2
		lineLeft := effect.Offset.X - effect.LineWidth/2
		sys.drawHorizontalLine(world, buf, lineLeft, lineY, int(effect.LineWidth), effect.Color)
	}

	// バッファ全体にアルファを適用して画面に合成する
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(effect.Alpha))
	screen.DrawImage(buf, op)
}

// drawDamageText はエンティティの立て板の頭にダメージテキストを描画する。
// Offset は投影後の画面座標へ足す。浮き上がる量は演出であって世界の長さではないので、
// カメラの遠近やズームで速さが変わらないようにする。
func (sys *VisualEffectSystem) drawDamageText(screen *ebiten.Image, projector render3d.Projector, face text.Face, gridElement *gc.GridElement, effect *gc.DamageTextEffect) {
	anchor, ok := projector.BillboardTop(gridElement.Coord)
	if !ok {
		return
	}
	screenX := float64(anchor.X) + effect.Offset.X
	screenY := float64(anchor.Y) + effect.Offset.Y

	// テキストサイズを測定して中央揃え
	textWidth, _ := text.Measure(effect.Text, face, 0)
	x := screenX - textWidth/2
	y := screenY

	// 透明度を適用した色
	alpha := uint8(effect.Alpha * 255)
	textColor := color.RGBA{effect.Color.R, effect.Color.G, effect.Color.B, alpha}
	outlineColor := color.RGBA{0, 0, 0, alpha}

	// アウトライン付きテキストを描画
	hud.OutlinedText(screen, effect.Text, face, x, y, textColor, outlineColor)
}

// drawHorizontalLine は両端がグラデーションで透明になる水平線を描画する
func (sys *VisualEffectSystem) drawHorizontalLine(world w.World, screen *ebiten.Image, x, y float64, width int, clr color.RGBA) {
	if width <= 0 {
		return
	}

	img := world.Resources.UIResources.GradientLine
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	srcWidth := float64(img.Bounds().Dx())
	op.GeoM.Scale(float64(width)/srcWidth, 1)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	screen.DrawImage(img, op)
}

// drawSpriteFadeoutEffect はスプライトの白シルエットフェードアウトエフェクトを描画する
func (sys *VisualEffectSystem) drawSpriteFadeoutEffect(world w.World, screen *ebiten.Image, projector render3d.Projector, gridElement *gc.GridElement, effect *gc.SpriteFadeoutEffect) error {
	if effect.Alpha <= 0 {
		return nil
	}
	if world.Resources.SpriteSheets == nil {
		return nil
	}

	// シェーダーを初期化（初回のみ）
	if sys.silhouetteShader == nil {
		shaderSource, err := assets.FS.ReadFile("file/shaders/white_silhouette.kage")
		if err != nil {
			return err
		}
		sys.silhouetteShader, err = ebiten.NewShader(shaderSource)
		if err != nil {
			return err
		}
	}

	// スプライトシートを取得
	spriteSheet, exists := world.Resources.SpriteSheets[effect.SpriteSheetName]
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
	img := gc.SubImage(texture.Image, image.Rect(left, top, right, bottom))

	// 立て板と同じ位置・大きさで重ねる。スプライトの高さが立て板1枚分になるよう拡大率を決め、
	// 立て板の中心へ合わせる
	scale, ok := projector.BillboardScale(gridElement.Coord)
	if !ok {
		return nil
	}
	center, ok := projector.TileCenter(gridElement.Coord, render3d.BillboardHeight/2)
	if !ok {
		return nil
	}
	// 高さ0のスプライトは拡大率が定まらず描けない。ゼロ除算を避けて描画をやめる
	if sprite.Height == 0 {
		return nil
	}
	zoom := scale / float64(sprite.Height)

	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(float64(-sprite.Width)/2, float64(-sprite.Height)/2)
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(float64(center.X), float64(center.Y))

	// ソース画像を設定
	op.Images[0] = img

	// 透明度をシェーダーに渡す（ColorScaleのAlphaを使用）
	op.ColorScale.ScaleAlpha(float32(effect.Alpha))

	// シェーダーで白シルエットを描画
	screen.DrawRectShader(sprite.Width, sprite.Height, sys.silhouetteShader, op)
	return nil
}
