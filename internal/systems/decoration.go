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

// 装飾は3Dシーンへ重ねる小さな絵で、寿命の持ち方が2種類ある。
//   - 状態従属・視点相対のヒント: 収納マーカーのように、ワールド状態から毎フレーム導出して出す。
//     エンティティ化せず collectDecorations で描画クアッドを直接積む。プレイヤーの移動で
//     出没するのでエンティティにすると生成削除が絶えず、視点状態が ECS に漏れるため導出のままにする。
//   - イベントで生じて残す物: 血痕・破片・撃破エフェクトのように、生成時に SpriteRender と
//     GridElement を持つエンティティを spawn する。collectBillboards と VisualEffectSystem が描く。
//     こちらはエンティティ自身が寿命を持つのでここでは扱わない。
//
// appendTileBillboard は両者に共通する「升にカメラ正面の板を1枚立てる」純幾何で、導出系はこれを通す。

// itemMarkerProximity はマーカーを出すプレイヤーからの近接距離。2マス以内。
// 少し離れていても、漁る価値のある升が見えるようにする。
const itemMarkerProximity = 2

// collectDecorations は状態従属・視点相対の装飾クアッドを quads へ足す。今は収納マーカーだけを扱う。
// マーカーはプレイヤー近接かつ視界内の升に、開ける前には見えない中身があることを示す。対象は
// 「中身のある収納」か「拾えるアイテムが2個以上重なった升」。単品で見えているアイテムは自前スプライトで
// 分かるので出さない。重なって隠れた分だけを指す。
func (sys *Render3DSystem) collectDecorations(world w.World, quads []r3quad, projector render3d.Projector) []r3quad {
	player, err := query.GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(player) {
		return quads
	}
	pc := world.Components.GridElement.Get(player).Coord
	near := func(c consts.Coord[consts.Tile]) bool {
		return geometry.ChebyshevDistance(pc, c) <= itemMarkerProximity
	}

	// 拾えるフィールドアイテムを升ごとに数える。近接分だけでよい。
	// Fixed でないフィールド物が拾える物なので、フィルタで Fixed を除く
	itemCount := map[consts.Coord[consts.Tile]]int{}
	itemQuery := query.ActiveFilter2[gc.LocationOnField, gc.GridElement](world).Without(ecs.C[gc.Fixed]()).Query()
	for itemQuery.Next() {
		e := itemQuery.Entity()
		c := world.Components.GridElement.Get(e).Coord
		if near(c) {
			itemCount[c]++
		}
	}

	// 中身のある収納の升を集める
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

	// マーカー画像は実際に1枚でも要るときだけ焼く。近くに対象が無い升配置では
	// フォントリソースに触れずに済み、リソースを持たない最小 world でも通る
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
			// 視界内の升にしか出さないので満照 tint で浮かせ、埋もれさせない
			height := render3d.BillboardHeight * itemMarkerHeightRatio
			quads = sys.appendTileBillboard(quads, projector, c, img, iw, ih, height, [3]float64{1, 1, 1})
		}
	}
	return quads
}

// itemMarkerGlyph はマーカーに出す汎用記号。中身や重なりがあることの合図。
const itemMarkerGlyph = "!"

// itemMarkerHeightRatio はマーカーの高さをビルボード高の何割にするか。ズームや奥行きで
// ビルボードが伸縮しても比率を保ち、常に同じ大きさに見せる。
const itemMarkerHeightRatio = 0.65

var (
	itemMarkerImg     *ebiten.Image
	itemMarkerImgOnce sync.Once
)

// itemMarkerImage は縁取り済みの記号を1枚の画像へ焼いて返す。text 描画は共有グリフキャッシュを
// 触るので一度だけ行い、以降はこの画像を拡大して重ねる。拡大縮小しても縁取りごと比率が保たれる。
func itemMarkerImage(face text.Face) *ebiten.Image {
	itemMarkerImgOnce.Do(func() {
		gw, gh := uicore.MeasureText(itemMarkerGlyph, face)
		// 縁取りを太めに敷いて、細い記号でも太く見えるようにする
		const outline = 3
		const pad = outline + 1
		// 緑のアイテムとも黒縁で分離される蛍光緑にして、どの升にマーカーが付くかを目立たせる
		markerColor := color.RGBA{R: 60, G: 255, B: 90, A: 255}
		img := ebiten.NewImage(gw+pad*2, gh+pad*2)
		cv := uicore.NewEbitenCanvas(img)
		// 暗色を周囲 outline 半径へずらして重ね、太い縁取りにする。中央に本体を重ねる
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
	// 右上隅を基準に、左と下へ板を広げる。幅は画像縦横比で決める
	fw := height * float64(iw) / float64(ih)
	tr := corner
	tl := render3d.Add(corner, render3d.Scale(right, -fw))
	br := render3d.Add(corner, render3d.Scale(up, -height))
	bl := render3d.Add(tl, render3d.Scale(up, -height))
	sys.addQuad(&quads, tl, tr, br, bl, img, 0, 0, float64(iw), float64(ih), col)
	return quads
}
