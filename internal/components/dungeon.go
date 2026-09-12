package components

import (
	"image/color"

	"github.com/kijimaD/ruins/internal/consts"
)

// TileIdx はタイル番号
type TileIdx int

// LightInfo は視界内タイルの光源情報を保持する
type LightInfo struct {
	Darkness float64
	Color    color.RGBA
}

// SeamlessBand はオーバーワールドのアクティブ帯の永続状態を保持する。
// Active が true のときのみ有効。全フィールドがスカラーなので serde に乗る。
// これによりロード後や遺跡遷移後に Band を再構築できる。
type SeamlessBand struct {
	// Active はシームレスワールド中かを表す
	Active bool
	// RunSeed はチャンク決定的生成の元 seed
	RunSeed uint64
	// NorthIndex は北進したチャンク数。北へ進むほど増える
	NorthIndex consts.Chunk
	// ChunkW は1チャンクの幅
	ChunkW consts.Tile
	// ChunkH は帯の高さ
	ChunkH consts.Tile
	// Cols は帯の横のチャンク列数。有界
	Cols consts.Chunk
	// Rows は帯の縦チャンク行数。ゼロ値なら復元時に 1 へ正規化する
	Rows consts.Chunk
}

// BandOriginY は帯ローカル Y=0 すなわち北端が指す絶対タイル Y。北は -Y なので NorthIndex ぶん負へ伸びる。
func (sb SeamlessBand) BandOriginY() consts.AbsTileY {
	return consts.AbsTileY(-sb.NorthIndex.Tiles(sb.ChunkH))
}

// LocalToAbsY は帯ローカル Y を絶対 Y に変換する。
func (sb SeamlessBand) LocalToAbsY(localY consts.Tile) consts.AbsTileY {
	return consts.AbsTileY(localY) + sb.BandOriginY()
}

// Dungeon は現在地を指すシングルトン。共存する複数ステージのうち、今どれが稼働中かを指す
// identity だけを持つ。フィールド寸法・探索履歴・帯データなどステージ固有の状態は各ステージの
// StageField が、時間や視界などグローバルな状態は専用シングルトンが持つ。
type Dungeon struct {
	// CurrentStage は現在稼働しているステージのキー。往復の swap で切り替える。
	// 階層数は CurrentStage.Depth から、ダンジョン定義名は CurrentStage.Name から導出する。
	// オーバーワールドは深度0で NewOverworldStage() の固定名を持つ。フィールド寸法・探索履歴・帯データは
	// 各ステージの StageField が持ち、ここは identity だけを指す。
	CurrentStage StageKey
}

// NewDungeon は初期化されたDungeonを返す
func NewDungeon() *Dungeon {
	return &Dungeon{}
}

// Level は現在の階層
// タイル計算メソッドを提供する
type Level struct {
	// 横のタイル数
	TileWidth consts.Tile
	// 縦のタイル数
	TileHeight consts.Tile
}

// CoordToIndex はタイル座標から、タイルスライスのインデックスを求める
func (l *Level) CoordToIndex(pos consts.Coord[consts.Tile]) TileIdx {
	return TileIdx(int(pos.Y)*int(l.TileWidth) + int(pos.X))
}

// IndexToCoord はタイルスライスのインデックスからタイル座標を求める。CoordToIndex の逆操作
func (l *Level) IndexToCoord(idx TileIdx) consts.Coord[consts.Tile] {
	x := consts.Tile(int(idx) % int(l.TileWidth))
	y := consts.Tile(int(idx) / int(l.TileWidth))

	return consts.Coord[consts.Tile]{X: x, Y: y}
}

// Width はステージ幅。横の全体ピクセル数
func (l *Level) Width() consts.WorldPixel {
	return consts.WorldPixel(int(l.TileWidth) * int(consts.TileSize))
}

// Height はステージ縦。縦の全体ピクセル数
func (l *Level) Height() consts.WorldPixel {
	return consts.WorldPixel(int(l.TileHeight) * int(consts.TileSize))
}
