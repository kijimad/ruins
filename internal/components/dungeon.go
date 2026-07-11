package components

import (
	"github.com/kijimaD/ruins/internal/consts"
)

// TileIdx はタイル番号
type TileIdx int

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
}

// NewDungeon は初期化されたDungeonを返す
func NewDungeon() *Dungeon {
	return &Dungeon{
		ExploredTiles: make(map[GridElement]bool),
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
