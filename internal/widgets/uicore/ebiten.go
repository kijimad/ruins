package uicore

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// EbitenCanvas は Canvas を ebiten の描画で満たす本番実装。
type EbitenCanvas struct {
	screen *ebiten.Image
}

var _ Canvas = (*EbitenCanvas)(nil)

// NewEbitenCanvas は描画先スクリーンを与えて Canvas を作る。
func NewEbitenCanvas(screen *ebiten.Image) *EbitenCanvas {
	return &EbitenCanvas{screen: screen}
}

// FillRect は EbitenCanvas を実装する。opts に正の Radius があれば四隅を丸めて塗る。
func (e *EbitenCanvas) FillRect(r image.Rectangle, c color.Color, opts RectOptions) {
	if opts.Radius > 0 {
		e.drawShape(r.Min, roundedFillShape(r.Dx(), r.Dy(), opts.Radius, c))
		return
	}
	vector.FillRect(e.screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
}

// StrokeRect は EbitenCanvas を実装する。opts に正の Radius があれば四隅を丸めて枠を描く。
func (e *EbitenCanvas) StrokeRect(r image.Rectangle, width int, c color.Color, opts RectOptions) {
	if opts.Radius > 0 {
		// 枠は線幅の半分だけ矩形の外へ出る。画像は線幅ぶん広く焼いてあるので、左上を線幅ぶん戻して重ねる
		shape := roundedStrokeShape(r.Dx(), r.Dy(), width, opts.Radius, c)
		e.drawShape(r.Min.Sub(image.Pt(width, width)), shape)
		return
	}
	vector.StrokeRect(e.screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), float32(width), c, false)
}

// drawShape は焼き済みの形状画像を pos へ素で描く。色は焼き込み済みなので色掛けはしない。
func (e *EbitenCanvas) drawShape(pos image.Point, shape *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(pos.X), float64(pos.Y))
	e.screen.DrawImage(shape, op)
}

// clampRadius は半径を矩形の短辺の半分までに抑える。小さな矩形へ大きな半径を与えても
// 円弧が交差してパスが壊れないようにする。
func clampRadius(radius, w, h int) float32 {
	rad := float32(radius)
	if rad > float32(w)/2 {
		rad = float32(w) / 2
	}
	if rad > float32(h)/2 {
		rad = float32(h) / 2
	}
	return rad
}

// roundedRectPath は四隅を半径 radius で丸めた矩形のパスを組む。半径は矩形の短辺の半分までに丸める。
func roundedRectPath(r image.Rectangle, radius int) vector.Path {
	x, y := float32(r.Min.X), float32(r.Min.Y)
	w, h := float32(r.Dx()), float32(r.Dy())
	rad := clampRadius(radius, r.Dx(), r.Dy())
	var p vector.Path
	p.MoveTo(x+rad, y)
	p.LineTo(x+w-rad, y)
	p.ArcTo(x+w, y, x+w, y+rad, rad)
	p.LineTo(x+w, y+h-rad)
	p.ArcTo(x+w, y+h, x+w-rad, y+h, rad)
	p.LineTo(x+rad, y+h)
	p.ArcTo(x, y+h, x, y+h-rad, rad)
	p.LineTo(x, y+rad)
	p.ArcTo(x, y, x+rad, y, rad)
	p.Close()
	return p
}

// 角丸は毎フレーム vector.FillPath でラスタライズすると CPU 実描画で重く、パスの分割で
// フレームごとにアロケーションも出る。同じ寸法・色は繰り返し使われるので、形状を色ごと画像へ
// 一度だけ焼き、描画は素の DrawImage で済ませる。
// 焼く画像は Unmanaged にして共有アトラスへ載せない。アトラス配置は生成順に依存し、並行する
// ゴールデンテストで描画結果がぶれる。Unmanaged なら各形状が独立テクスチャで決定的になる。
// 色を描画時に掛けると半透明色で量子化が二重になり直接描画とずれるため、色は焼く時点で塗り込む。
// Draw は単一ゴルーチンだがテストが並行にキャッシュへ触れるので map は mutex で守る。生成も
// ロック内に置き、並行テストで ebiten の画像生成が同時に走らないようにする。二重チェックで
// ロック外へ出すと生成が並行しうるので、あえて直列化する。
// キーはパネルの数種の寸法と theme の色に限られ有限なので、測定キャッシュと違い上限は要らない。
var (
	roundedShapeMu      sync.Mutex
	roundedFillShapes   = map[roundedFillKey]*ebiten.Image{}
	roundedStrokeShapes = map[roundedStrokeKey]*ebiten.Image{}
)

type roundedFillKey struct {
	w, h, radius int
	c            color.RGBA
}
type roundedStrokeKey struct {
	w, h, width, radius int
	c                   color.RGBA
}

// roundedFillShape は w×h の角丸塗りを色 c で焼いた画像を返す。左上を原点として描く。
func roundedFillShape(w, h, radius int, c color.Color) *ebiten.Image {
	rgba, _ := color.RGBAModel.Convert(c).(color.RGBA)
	key := roundedFillKey{w, h, radius, rgba}
	roundedShapeMu.Lock()
	defer roundedShapeMu.Unlock()
	if img, ok := roundedFillShapes[key]; ok {
		return img
	}
	img := ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{Unmanaged: true})
	p := roundedRectPath(image.Rect(0, 0, w, h), radius)
	dop := &vector.DrawPathOptions{AntiAlias: true}
	dop.ColorScale.ScaleWithColor(c)
	vector.FillPath(img, &p, &vector.FillOptions{}, dop)
	roundedFillShapes[key] = img
	return img
}

// roundedStrokeShape は w×h の角丸枠を色 c で焼いた画像を返す。枠は線幅の半分だけ矩形の外へ
// はみ出すので、線幅ぶん広げた画像に焼く。左上へ合わせるずらしは呼び出し側が線幅から決める。
func roundedStrokeShape(w, h, width, radius int, c color.Color) *ebiten.Image {
	rgba, _ := color.RGBAModel.Convert(c).(color.RGBA)
	key := roundedStrokeKey{w, h, width, radius, rgba}
	roundedShapeMu.Lock()
	defer roundedShapeMu.Unlock()
	if img, ok := roundedStrokeShapes[key]; ok {
		return img
	}
	img := ebiten.NewImageWithOptions(image.Rect(0, 0, w+width*2, h+width*2), &ebiten.NewImageOptions{Unmanaged: true})
	p := roundedRectPath(image.Rect(width, width, width+w, width+h), radius)
	dop := &vector.DrawPathOptions{AntiAlias: true}
	dop.ColorScale.ScaleWithColor(c)
	vector.StrokePath(img, &p, &vector.StrokeOptions{Width: float32(width)}, dop)
	roundedStrokeShapes[key] = img
	return img
}

// FillTriangle は EbitenCanvas を実装する。3頂点の三角形を塗る。text/v2 を通らないのでロックは要らない。
// 頂点が呼び出しごとに変わり形が定まらないので、角丸のような形状キャッシュは持たず毎回パスを組む。
func (e *EbitenCanvas) FillTriangle(p0, p1, p2 [2]float32, c color.Color) {
	var path vector.Path
	path.MoveTo(p0[0], p0[1])
	path.LineTo(p1[0], p1[1])
	path.LineTo(p2[0], p2[1])
	path.Close()
	dop := &vector.DrawPathOptions{AntiAlias: true}
	dop.ColorScale.ScaleWithColor(c)
	vector.FillPath(e.screen, &path, &vector.FillOptions{}, dop)
}

// DrawText は EbitenCanvas を実装する。pos を左上として1行を描く。
func (e *EbitenCanvas) DrawText(pos image.Point, s string, face text.Face, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(pos.X), float64(pos.Y))
	op.ColorScale.ScaleWithColor(c)
	// ebiten text/v2 の共有グリフキャッシュを壊さないよう描画を直列化する。詳細は textMu を参照
	textMu.Lock()
	text.Draw(e.screen, s, face, op)
	textMu.Unlock()
}

// DrawImage は EbitenCanvas を実装する。pos を左上として画像を描く。
func (e *EbitenCanvas) DrawImage(pos image.Point, img *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(pos.X), float64(pos.Y))
	e.screen.DrawImage(img, op)
}

// DrawImageRect は EbitenCanvas を実装する。dst に収まるよう縦横比を保って縮小し、左寄せ・縦中央で描く。
func (e *EbitenCanvas) DrawImageRect(dst image.Rectangle, img *ebiten.Image) {
	if img == nil {
		return
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw <= 0 || ih <= 0 || dst.Dx() <= 0 || dst.Dy() <= 0 {
		return
	}
	scale := math.Min(math.Min(float64(dst.Dx())/float64(iw), float64(dst.Dy())/float64(ih)), 1)
	dh := float64(ih) * scale
	// 左寄せ・縦中央。アイコンとキーキャップを列の左に揃え、行の高さの中央へ置く
	ox := float64(dst.Min.X)
	oy := float64(dst.Min.Y) + (float64(dst.Dy())-dh)/2
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(ox, oy)
	// 収まりきらない画像だけがここで縮む。端数倍率の nearest はテクセル境界の画素が
	// 揺れるので linear にする。一覧のアイコンは表示サイズで渡ってくるため等倍で通り、
	// この分岐には入らない
	if scale != 1 {
		op.Filter = ebiten.FilterLinear
	}
	e.screen.DrawImage(img, op)
}

// DrawImageTintedRect は EbitenCanvas を実装する。img を dst いっぱいに引き伸ばし tint を掛けて描く。
func (e *EbitenCanvas) DrawImageTintedRect(dst image.Rectangle, img *ebiten.Image, tint color.Color) {
	if img == nil {
		return
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 || dst.Dx() <= 0 || dst.Dy() <= 0 {
		return
	}
	sx, sy := float64(dst.Dx())/float64(b.Dx()), float64(dst.Dy())/float64(b.Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(float64(dst.Min.X), float64(dst.Min.Y))
	op.ColorScale.ScaleWithColor(tint)
	// 伸ばす軸がどれも2テクセル以上あるときだけ linear にする。判定の理由は stretchNeedsBlend を参照
	if stretchNeedsBlend(sx, b.Dx()) && stretchNeedsBlend(sy, b.Dy()) {
		op.Filter = ebiten.FilterLinear
	}
	e.screen.DrawImage(img, op)
}

// stretchNeedsBlend はその軸を混ぜるべきかを返す。端数倍率の nearest はテクセル境界の
// 画素が揺れるので、混ぜられる軸は linear で混ぜたい。等倍なら境界をまたがず混ぜる必要が無い。
// 1テクセルしか無い軸は混ぜる相手が画像の外側の透明しかなく、端がぼやけるだけなので混ぜない。
func stretchNeedsBlend(scale float64, srcLen int) bool {
	return scale != 1 && srcLen > 1
}

// DrawNineSlice は EbitenCanvas を実装する。ソースを縦横3分割し、四隅は原寸、辺は片方向、
// 中央は両方向へ伸ばして9セルを dst へ描く。
func (e *EbitenCanvas) DrawNineSlice(dst image.Rectangle, img *ebiten.Image, bx, by [3]int) {
	if img == nil {
		return
	}
	// ソースの分割境界
	sx := [4]int{0, bx[0], bx[0] + bx[1], bx[0] + bx[1] + bx[2]}
	sy := [4]int{0, by[0], by[0] + by[1], by[0] + by[1] + by[2]}
	// dst の分割境界。四隅は原寸、中央は残りを埋める
	midW := max(dst.Dx()-bx[0]-bx[2], 0)
	midH := max(dst.Dy()-by[0]-by[2], 0)
	dx := [4]int{dst.Min.X, dst.Min.X + bx[0], dst.Min.X + bx[0] + midW, dst.Max.X}
	dy := [4]int{dst.Min.Y, dst.Min.Y + by[0], dst.Min.Y + by[0] + midH, dst.Max.Y}

	for r := range 3 {
		for c := range 3 {
			sw, sh := sx[c+1]-sx[c], sy[r+1]-sy[r]
			dw, dh := dx[c+1]-dx[c], dy[r+1]-dy[r]
			if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
				continue
			}
			sub, ok := img.SubImage(image.Rect(sx[c], sy[r], sx[c+1], sy[r+1])).(*ebiten.Image)
			if !ok {
				continue
			}
			ssx, ssy := float64(dw)/float64(sw), float64(dh)/float64(sh)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(ssx, ssy)
			op.GeoM.Translate(float64(dx[c]), float64(dy[r]))
			// 四隅は原寸で通り、伸びるのは辺と中央だけ。端数倍率の nearest は
			// テクセル境界の画素が揺れるので linear にする
			if ssx != 1 || ssy != 1 {
				op.Filter = ebiten.FilterLinear
			}
			e.screen.DrawImage(sub, op)
		}
	}
}
