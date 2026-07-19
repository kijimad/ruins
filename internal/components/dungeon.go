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
	// EastIndex は東進したチャンク数
	EastIndex consts.Chunk
	// ChunkW は1チャンクの幅
	ChunkW consts.Tile
	// ChunkH は帯の高さ
	ChunkH consts.Tile
	// K は帯のチャンク数
	K consts.Chunk

	// Front は寒波前線の永続状態。帯の Active に従属し、帯とセットで復元される。
	Front SeamlessFront
}

// SeamlessFront は寒波前線の永続状態。現在位置は保存せず、config と永続の
// GameTime.TotalTurns から決定的に導出する。位置を持たないのでドリフトせず、ロードでも自然に復元される。
//
// ジオメトリは温度・移動・描画の各消費者が同じ半開区間を使うようここに集約する。
// worldstream.Front と同じ意味・同じメソッド面だが、あちらは components を import するため
// systems/activity から使えない。永続スカラーの上でこちらを正典にする。
type SeamlessFront struct {
	// Active は寒波前線が有効か
	Active bool
	// StartAbsX はラン開始時の極低温ゾーン東端の絶対タイルX。ローカルでなく絶対軸
	StartAbsX consts.AbsTileX
	// ColdWidth は極低温ゾーンの幅
	ColdWidth consts.Tile
	// AdvanceTurns はこの経過ターンごとに Step タイル東進する
	AdvanceTurns consts.Turn
	// Step は1回の前進量
	Step consts.Tile
	// EastAbsX は現在の極低温ゾーン東端の絶対タイルX。config と総ターン数から毎ターン導出した
	// 現在位置のキャッシュ。描画や凍結効果など後続の消費者がここを読む
	EastAbsX consts.AbsTileX
}

// BandOriginX は帯ローカル X=0 が指す絶対タイル X。
func (sb SeamlessBand) BandOriginX() consts.AbsTileX {
	return consts.AbsTileX(sb.EastIndex.Tiles(sb.ChunkW))
}

// LocalToAbsX は帯ローカル X を絶対 X に変換する。
func (sb SeamlessBand) LocalToAbsX(localX consts.Tile) consts.AbsTileX {
	return consts.AbsTileX(localX) + sb.BandOriginX()
}

// ColdZoneWest は極低温ゾーン西端＝破棄/進入不可ラインの絶対 X。
func (f SeamlessFront) ColdZoneWest() consts.AbsTileX {
	return f.EastAbsX - consts.AbsTileX(f.ColdWidth)
}

// InColdZone は絶対 X が極低温ゾーン (ColdZoneWest, EastAbsX] 内かを返す。西端は含まない。
func (f SeamlessFront) InColdZone(absX consts.AbsTileX) bool {
	return absX > f.ColdZoneWest() && absX <= f.EastAbsX
}

// IsWestOfFront は絶対 X が進入不可ライン、すなわち極低温ゾーン西端以西かを返す。
func (f SeamlessFront) IsWestOfFront(absX consts.AbsTileX) bool {
	return absX <= f.ColdZoneWest()
}

// Dungeon は冒険出発から帰還までを1セットとした情報を保持する。
// 冒険出発から帰還までは複数階層が存在し、複数階層を通しての情報を保持する必要がある。
type Dungeon struct {
	// 現在階のフィールド情報
	Level Level
	// 階層数
	Depth int
	// 探索済みタイルのマップ。座標をキーとして使用。
	// GridElement(struct)キーのためserde不可、ロード時に再構築する
	ExploredTiles map[GridElement]bool `json:"-"`
	// ミニマップの設定
	MinimapSettings MinimapSettings
	// 視界を更新するか外部から設定するフラグ
	NeedsForceUpdate bool
	// DefinitionName はダンジョン定義名
	DefinitionName string
	// GameTime はゲーム内時間を保持する
	GameTime GameTime
	// SelectedWeaponSlot は選択中の武器スロット番号（1-5）
	SelectedWeaponSlot int
	// VisibleTiles は現在フレームで実際に見えているタイルのマップ。毎フレーム更新される。
	// GridElement(struct)キーのためserde不可、毎フレーム再構築される
	VisibleTiles map[GridElement]bool `json:"-"`
	// LightSourceCache は視界内タイルの光源情報。VisionSystemが計算し描画側が参照する。
	// 視界更新のたびに再構築されるためserde不可
	LightSourceCache map[GridElement]LightInfo `json:"-"`
	// SeamlessBand はシームレスワールドの帯の永続状態。通常ダンジョンでは Active=false
	SeamlessBand SeamlessBand
}

// NewDungeon は初期化されたDungeonを返す
func NewDungeon() *Dungeon {
	return &Dungeon{
		ExploredTiles:    make(map[GridElement]bool),
		VisibleTiles:     make(map[GridElement]bool),
		LightSourceCache: make(map[GridElement]LightInfo),
		MinimapSettings: MinimapSettings{
			Width:   150,
			Height:  150,
			OffsetX: 10,
			OffsetY: 10,
			Scale:   3,
		},
		SelectedWeaponSlot: 1,
	}
}

// Level は現在の階層
// タイル計算メソッドを提供する
type Level struct {
	// 横のタイル数
	TileWidth consts.Tile
	// 縦のタイル数
	TileHeight consts.Tile
}

// XYTileIndex はタイル座標から、タイルスライスのインデックスを求める
func (l *Level) XYTileIndex(tx consts.Tile, ty consts.Tile) TileIdx {
	return TileIdx(int(ty)*int(l.TileWidth) + int(tx))
}

// XYTileCoord はタイルスライスのインデックスからタイル座標を求める
func (l *Level) XYTileCoord(idx TileIdx) (consts.Pixel, consts.Pixel) {
	x := int(idx) % int(l.TileWidth)
	y := int(idx) / int(l.TileWidth)

	return consts.Pixel(x), consts.Pixel(y)
}

// Width はステージ幅。横の全体ピクセル数
func (l *Level) Width() consts.Pixel {
	return consts.Pixel(int(l.TileWidth) * int(consts.TileSize))
}

// Height はステージ縦。縦の全体ピクセル数
func (l *Level) Height() consts.Pixel {
	return consts.Pixel(int(l.TileHeight) * int(consts.TileSize))
}

// MinimapSettings はミニマップの設定を管理する
type MinimapSettings struct {
	// ミニマップのサイズ（ピクセル単位）
	Width  int
	Height int
	// ミニマップの表示位置（画面右上に配置）
	OffsetX int
	OffsetY int
	// ミニマップのスケール（何ピクセルで1タイルを表すか）
	Scale int
}
