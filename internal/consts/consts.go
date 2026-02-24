package consts

// ========== 基本型 ==========

// Pixel はピクセル単位。計算用にfloat64
type Pixel float64

// Tile はタイルの位置。ピクセル数ではない
type Tile int

// ========== ウィンドウサイズ ==========

const (
	// MinGameWidth はゲームウィンドウの最小幅
	MinGameWidth = 960
	// MinGameHeight はゲームウィンドウの最小高さ
	MinGameHeight = 720
)

// ========== ゲーム定数 ==========

const (
	// TileSize はタイルの寸法
	TileSize Pixel = 32
	// MapTileWidth はマップの横タイル数
	MapTileWidth = 50
	// MapTileHeight はマップの縦タイル数
	MapTileHeight = 50
	// GameClearDepth はゲームクリアとなる深度
	GameClearDepth = 100
	// VisionRadiusTiles は視界半径（タイル単位）
	VisionRadiusTiles = 16
)

// ========== タイル名 ==========

const (
	// TileNameWall は壁タイル名
	TileNameWall = "wall"
	// TileNameFloor は床タイル名
	TileNameFloor = "floor"
	// TileNameVoid は虚空タイル名
	TileNameVoid = "void"
)
