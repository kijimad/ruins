package systems

import (
	"image"
	"image/color"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 状態従属・視点相対のヒントはここで毎フレーム導出して描く。プレイヤーの移動で出没するので
// エンティティ化すると生成削除が絶えず視点状態が ECS に漏れるため、導出のまま扱う。イベントで生じて
// 残す装飾は血痕・破片・撃破エフェクトのように SpriteRender と GridElement を持つエンティティを
// spawn し、collectBillboards と VisualEffectSystem が描く。

// itemMarkerProximity はマーカーを出すプレイヤーからの近接距離。少し離れていても漁る価値のある升を見せる。
const itemMarkerProximity = 2

// collectDecorations は状態従属の装飾クアッドを quads へ足す。今は収納マーカーだけを扱う。対象は
// 「中身のある収納」か「拾えるアイテムが2個以上重なった升」。単品で見えているアイテムは自前スプライトで
// 分かるので出さず、重なって隠れた分だけを指す。
func (sys *Render3DSystem) collectDecorations(world w.World, quads []r3quad, projector render3d.Projector) []r3quad {
	player, err := query.GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(player) {
		return quads
	}
	pc := world.Components.GridElement.Get(player).Coord
	near := func(c consts.Coord[consts.Tile]) bool {
		return geometry.ChebyshevDistance(pc, c) <= itemMarkerProximity
	}

	// 拾えるフィールドアイテムを升ごとに数える。Fixed でないフィールド物が拾える物なので Fixed を除く
	itemCount := map[consts.Coord[consts.Tile]]int{}
	itemQuery := query.ActiveFilter2[gc.LocationOnField, gc.GridElement](world).Without(ecs.C[gc.Fixed]()).Query()
	for itemQuery.Next() {
		e := itemQuery.Entity()
		c := world.Components.GridElement.Get(e).Coord
		if near(c) {
			itemCount[c]++
		}
	}

	storageHas := map[consts.Coord[consts.Tile]]bool{}
	stQuery := query.ActiveFilter2[gc.Interactable, gc.GridElement](world).Query()
	for stQuery.Next() {
		e := stQuery.Entity()
		c := world.Components.GridElement.Get(e).Coord
		if !near(c) {
			continue
		}
		if !slices.Contains(world.Components.Interactable.Get(e).Interactions, gc.InteractionStorage) {
			continue
		}
		if query.HasStorageItems(world, e) {
			storageHas[c] = true
		}
	}

	// 1枚でも要るときだけ画像を焼く。対象が無ければフォントに触れず、UI リソース無しの world でも通る
	var img *ebiten.Image
	var iw, ih int
	for dy := -itemMarkerProximity; dy <= itemMarkerProximity; dy++ {
		for dx := -itemMarkerProximity; dx <= itemMarkerProximity; dx++ {
			c := consts.Coord[consts.Tile]{X: pc.X + consts.Tile(dx), Y: pc.Y + consts.Tile(dy)}
			if !storageHas[c] && itemCount[c] < 2 {
				continue
			}
			if !query.IsInVision(world, pc, c) {
				continue
			}
			if img == nil {
				img = itemMarkerImage(world.Resources.UIResources.Text.SplashFontFace)
				iw, ih = img.Bounds().Dx(), img.Bounds().Dy()
				if iw == 0 || ih == 0 {
					return quads
				}
			}
			// 視界内の升にしか出さないので満照 tint で浮かせる
			height := render3d.BillboardHeight * itemMarkerHeightRatio
			quads = sys.appendTileBillboard(quads, projector, c, img, iw, ih, height, [3]float64{1, 1, 1})
		}
	}
	return quads
}

// itemMarkerGlyph はマーカーに出す汎用記号。中身や重なりがあることの合図。
const itemMarkerGlyph = "!"

// itemMarkerHeightRatio はマーカー高をビルボード高の何割にするか。奥行きやズームで見かけの比率を保つ。
const itemMarkerHeightRatio = 0.65

var (
	itemMarkerImg     *ebiten.Image
	itemMarkerImgOnce sync.Once
)

// itemMarkerImage は縁取り済みの記号を1枚の画像へ焼いて返す。text 描画は共有グリフキャッシュを
// 触るので一度だけ行い、以降はこの画像を拡大して重ねる。
func itemMarkerImage(face text.Face) *ebiten.Image {
	itemMarkerImgOnce.Do(func() {
		gw, gh := uicore.MeasureText(itemMarkerGlyph, face)
		// 細い記号でも太く見えるよう縁取りを太めに敷く
		const outline = 3
		const pad = outline + 1
		// 緑のアイテムとも黒縁で分離される蛍光緑にして目立たせる
		markerColor := color.RGBA{R: 60, G: 255, B: 90, A: 255}
		img := ebiten.NewImage(gw+pad*2, gh+pad*2)
		cv := uicore.NewEbitenCanvas(img)
		// 暗色を周囲 outline 半径へずらして重ね、中央へ本体を重ねる
		for dy := -outline; dy <= outline; dy++ {
			for dx := -outline; dx <= outline; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				cv.DrawText(image.Pt(pad+dx, pad+dy), itemMarkerGlyph, face, theme.HUDTextOutline)
			}
		}
		cv.DrawText(image.Pt(pad, pad), itemMarkerGlyph, face, markerColor)
		itemMarkerImg = img
	})
	return itemMarkerImg
}

// appendTileBillboard は升のビルボード右上隅を基準に、画像1枚をカメラ正面の板として quads へ足す。
// ワールド空間の板なので emit の深度ソートで手前の壁に隠れる。height はビルボード高に対する実寸で、
// 投影後の見かけの大きさは奥行きやズームに追従する。col は乗算 tint。導出系の装飾はこの幾何を共有する。
func (sys *Render3DSystem) appendTileBillboard(quads []r3quad, projector render3d.Projector, c consts.Coord[consts.Tile], img *ebiten.Image, iw, ih int, height float64, col [3]float64) []r3quad {
	// collectBillboards と同じ幾何でビルボード右上隅の world 座標を組む
	const bw = 0.45
	right := projector.Right()
	up := render3d.At(0, 1, 0)
	corner := render3d.Add(
		render3d.Add(render3d.At(float64(c.X)+0.5, 0, float64(c.Y)+0.5), render3d.Scale(right, bw)),
		render3d.At(0, render3d.BillboardHeight, 0),
	)
	// 右上隅を基準に左と下へ板を広げる。幅は画像縦横比で決める
	fw := height * float64(iw) / float64(ih)
	tr := corner
	tl := render3d.Add(corner, render3d.Scale(right, -fw))
	br := render3d.Add(corner, render3d.Scale(up, -height))
	bl := render3d.Add(tl, render3d.Scale(up, -height))
	sys.addQuad(&quads, tl, tr, br, bl, img, 0, 0, float64(iw), float64(ih), col)
	return quads
}
