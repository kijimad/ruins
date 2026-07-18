package consts

import "github.com/mlange-42/ark/ecs"

// InvalidEntity はエラー時の戻り値として使うセンチネル値。
// Ark のゼロ値 Entity は無効なエンティティを表す。
var InvalidEntity = ecs.Entity{}

// ========== 基本型 ==========

// Pixel はピクセル単位。計算用にfloat64
type Pixel float64

// Tile はタイルの位置。ピクセル数ではない
type Tile int

// ========== ウィンドウサイズ ==========

const (
	// GameWidth はゲームの論理解像度の幅。Ebiten の Layout が返す固定値で、
	// ウィンドウはこのサイズを拡大縮小して表示する
	GameWidth = 960
	// GameHeight はゲームの論理解像度の高さ
	GameHeight = 720
)

// ========== アクションポイント ==========

const (
	// MinActionThreshold は行動可能な最小AP
	MinActionThreshold = 0
	// StandardActionCost は標準的なアクションのAPコスト
	StandardActionCost = 100
	// MinorActionCost は軽量アクションのAPコスト
	MinorActionCost = 50
	// DefaultPlayerMoves はプレイヤーの初期移動ポイント
	DefaultPlayerMoves = 100
)

// ========== ゲーム定数 ==========

const (
	// TileSize はタイルの寸法
	TileSize Pixel = 32
)

const (
	// MapTileWidth はマップの横タイル数
	MapTileWidth Tile = 50
	// MapTileHeight はマップの縦タイル数
	MapTileHeight Tile = 50
	// VisionRadiusTiles は視界半径（タイル単位）
	// 視界の境界が画面内に見えないようにする
	VisionRadiusTiles Tile = 24
)

const (
	// AIVisionDistance はAIエンティティの視界距離（タイル単位）
	AIVisionDistance Tile = 5
)

const (
	// GameClearDepth はゲームクリアとなる深度
	GameClearDepth = 100
)

// ========== タイル名 ==========

const (
	// TileNameWall は壁タイル名
	TileNameWall = "wall"
	// TileNameFloor は床タイル名
	TileNameFloor = "floor"
	// TileNameDirt は土のタイル名
	TileNameDirt = "dirt"
	// TileNameVoid は虚空タイル名
	TileNameVoid = "void"
)
