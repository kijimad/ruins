package states

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// OverworldMapState はオーバーワールドの種別俯瞰図を全画面で表示するステート。
// 1チャンク=1建物の縮尺に合わせ、各チャンクを1マスの色と文字で示す。1マス=1つの場所。
// 生成は純関数なので ECS のタイルを生成せずに算出して描ける。
type OverworldMapState struct {
	es.BaseState[w.World]

	view      overworld.MacroView        // プレイヤー中心のチャンク俯瞰。glyph 格子とマーカー
	playerAbs consts.Coord[consts.Chunk] // 現在地の絶対チャンク座標。ヘッダ表示に使う
	facing    gc.Orient                  // カメラの水平向き。現在地ポインタが指す方角
	cellPx    consts.ScreenPixel         // 1チャンクのセル寸法。表示範囲の半径の算出と描画で共有する
	body      uicore.Drawable            // モーダルのパネル。初回 Draw で1度組み以後描く
}

var _ es.State[w.World] = &OverworldMapState{}

// OnPause はステートが一時停止される際に呼ばれる。
func (st *OverworldMapState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる。
func (st *OverworldMapState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる。
func (st *OverworldMapState) OnStop(_ w.World) error { return nil }

// 全画面図のセル寸法・半径の範囲。帯が短いとセルが巨大化、広いと潰れるのを両側で防ぐ。
// セルを小さくするほどモーダル幅に多くのチャンクが収まり、見える範囲が広がる
const (
	overworldMapMinCell    = 10
	overworldMapMaxCell    = 18
	overworldMapMinRadius  = 3
	overworldMapMarginCols = 3 // セル幅算出で cols に足す左右の余白の列相当ぶん
)

// modalInner はモーダルパネルの内側矩形を返す。表示範囲の半径・セル寸法の算出とパネル画像の寸法で共有する。
func (st *OverworldMapState) modalInner(world w.World) image.Rectangle {
	return menuframe.PanelInner(menuframe.ModalRect(world))
}

// overworldMapCell は有界の cols 列がモーダル幅に収まるセル寸法を決める。左右の余白ぶんを足した
// 列数で内側幅を割り、上限下限で挟む。帯が細いとセルが巨大化するので上限で止める。
func overworldMapCell(inner image.Rectangle, cols consts.Chunk) consts.ScreenPixel {
	// cols<=0 の退化入力でもゼロ除算しないよう、関数内で 1 以上へ丸めてから割る
	denom := max(int(cols), 1) + overworldMapMarginCols
	cell := min(max(inner.Dx()/denom, overworldMapMinCell), overworldMapMaxCell)
	return consts.ScreenPixel(cell)
}

// overworldMapRadius はモーダル高さに収まるプレイヤー南北のチャンク数を返す。最低限は確保する。
// 地図の下に見出しと凡例を置くので、そのぶんの高さを差し引いてから収まる行数を出す。
func overworldMapRadius(inner image.Rectangle, cell consts.ScreenPixel) int {
	const headerLegendReserve = 160
	rows := (inner.Dy() - headerLegendReserve) / int(cell)
	return max((rows-1)/2, overworldMapMinRadius)
}

// OnStart はプレイヤー中心の地形俯瞰モデルを算出して保持する。表示中はプレイヤーが動かないため
// 一度だけ計算する。セル寸法と半径をモーダル寸法から決めてモーダルいっぱいに大きく見せ、モデル
// 生成は overworld.BuildMacroView に委ねる。フォグは HUD と同じで探索済みチャンクだけを開放する。
func (st *OverworldMapState) OnStart(world w.World) error {
	sb := query.GetSeamlessBand(world)
	if sb == nil || !sb.Active {
		return fmt.Errorf("overworld band is not valid")
	}
	playerTile, hasPlayer := query.PlayerBandTile(world)
	inner := st.modalInner(world)
	st.cellPx = overworldMapCell(inner, max(sb.Cols, 1))
	// 北進帯は縦に伸びるので、プレイヤーの絶対チャンク行を中心に近傍を見せる。
	// プレイヤー不在時は帯の中央行。絶対チャンク行への変換は SeamlessBand.AbsChunkRow に集約する
	centerRow := sb.AbsChunkRow((sb.Rows / 2).Tiles(sb.ChunkH))
	if hasPlayer {
		centerRow = sb.AbsChunkRow(playerTile.Y)
	}
	area := overworld.PlayerCenteredRange(centerRow, sb.Cols, overworldMapRadius(inner, st.cellPx))
	st.view = overworld.BuildMacroView(
		sb.RunSeed, sb.NorthIndex, sb.ChunkW, sb.ChunkH,
		area, playerTile, hasPlayer, query.DriveCubeTiles(world), query.DiscoveredChunks(world, sb),
	)

	// カメラ不在時は北を既定にする
	st.facing = 0
	if cam := query.GetPlayerCamera(world); cam != nil {
		st.facing = cam.Orient
	}

	// ヘッダ表示用の現在地の絶対チャンク座標。プレイヤーが居なければ -1 にして表示を空扱いにする
	st.playerAbs = consts.Coord[consts.Chunk]{X: -1}
	if hasPlayer {
		st.playerAbs = consts.Coord[consts.Chunk]{
			X: consts.Chunk(int(playerTile.X) / int(sb.ChunkW)),
			Y: sb.AbsChunkRow(playerTile.Y),
		}
	}
	return nil
}

// overworldMapBindings は種別俯瞰図の束縛表。開いたキーと同じ N でも閉じられる
var overworldMapBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
	{Key: ebiten.KeyN, Action: inputmapper.ActionCloseMenu},
}

// Update はキー入力で閉じるだけ。マップ表示中は時間を進めない。
func (st *OverworldMapState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := keybind.ReadInput(world, overworldMapBindings); ok && action == inputmapper.ActionCloseMenu {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	return st.ConsumeTransition(), nil
}

// Draw はモーダルのパネルを描く。他のメニュー系画面と同じく画面全体は塗らず、状態機械が
// 重ね描きする背後のダンジョンの上へパネルだけを重ねる。パネルの組み立ては UI リソースを要し
// モデル算出とは別なので、初回 Draw で1度だけ組んで以後使い回す。
func (st *OverworldMapState) Draw(world w.World, screen *ebiten.Image) error {
	if st.body == nil {
		st.body = st.buildBody(world)
	}
	st.body.Draw(uicore.NewEbitenCanvas(screen))
	return nil
}

// buildBody は俯瞰図を他メニューと同じモーダルのパネルとして組む。格子・凡例・見出し・マーカーを
// 1枚の画像へ描き、menuframe の PanelBG パネルへ収めて意匠を他モーダルへ揃える。
func (st *OverworldMapState) buildBody(world w.World) uicore.Drawable {
	res := world.Resources.UIResources
	rect := menuframe.ModalRect(world)
	// 画像は ImagePanel が画像を置く矩形と同じ寸法にする。PanelInner を両者で通し余白の値が
	// 変わってもずれないようにする
	inner := menuframe.PanelInner(rect)

	// 内容はパネル内側いっぱいの画像へ原点ローカルで描く。透明地なので描かない部分は
	// 背後のパネルテクスチャが透ける
	img := ebiten.NewImage(max(inner.Dx(), 1), max(inner.Dy(), 1))
	st.renderMap(world, img)

	return menuframe.ImagePanel(res, rect, img)
}

// renderMap は俯瞰図の見出し・格子・マーカー・凡例を dst へ原点ローカルで描く。格子と凡例の描画は
// hud に集約し、ここは見出しと配置の算出だけを持つ。
func (st *OverworldMapState) renderMap(world w.World, dst *ebiten.Image) {
	face := world.Resources.UIResources.Text.BodyFace
	// セルの記号は小さいセルへ収めるため小フォントにする。見出し・凡例は BodyFace のまま
	glyphFace := world.Resources.UIResources.Text.SmallFace

	// 文字も塗りも EbitenCanvas に集約する。text/v2 のグリフキャッシュは並行安全でなく、
	// 生の text.Draw はロックを迂回するため
	cv := uicore.NewEbitenCanvas(dst)

	cv.DrawText(image.Pt(8, 6), fmt.Sprintf("Overworld Map  Current Chunk %d, %d", st.playerAbs.X, st.playerAbs.Y), face, theme.TextPrimary)

	cell := st.cellPx
	// 格子は横をモーダル内側の中央へ寄せる
	cols := 0
	if len(st.view.Cells) > 0 {
		cols = len(st.view.Cells[0])
	}
	gridW := consts.ScreenPixel(cols) * cell
	originX := (consts.ScreenPixel(dst.Bounds().Dx()) - gridW) / 2
	if originX < 8 {
		originX = 8
	}
	const originY consts.ScreenPixel = 40
	// 格子・道・キューブ・現在地を hud.DrawMapGrid で描く。全画面図は常に記号を出す
	hud.DrawMapGrid(cv, st.view, hud.MapGridStyle{
		OriginX:      int(originX),
		OriginY:      int(originY),
		CellPx:       int(cell),
		MinGlyphPx:   0,
		GlyphFace:    glyphFace,
		MarkerFace:   face,
		PlayerFacing: st.facing,
	})

	hud.DrawMapLegend(cv, face, glyphFace, int(originY)+len(st.view.Cells)*int(cell)+16)
}
