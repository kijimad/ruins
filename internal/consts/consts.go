package consts

// ========== 基本型 ==========

// WorldPixel はワールド空間のピクセル単位。計算用に float64。
// フィールド上の絶対位置を表す。カメラ変換前の座標。
type WorldPixel float64

// ScreenPixel は画面空間のピクセル単位。カメラ変換後の描画位置を表す。
// WorldPixel とは別型にして、ワールド座標とスクリーン座標の取り違えを型で弾く。
// 変換は WorldToScreen で行う。
type ScreenPixel float64

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
	TileSize WorldPixel = 32
)

const (
	// MinimapWidth はミニマップの表示幅。ピクセル単位の固定 UI 値
	MinimapWidth = 150
	// MinimapHeight はミニマップの表示高さ
	MinimapHeight = 150
	// MinimapScale はミニマップのスケール。何ピクセルで1タイルを表すか
	MinimapScale = 3
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
	// TileNameWall は壁タイル名。プランナーが出力する論理名で、生成時のスプライトは TileNameDWall になる
	TileNameWall = "wall"
	// TileNameDWall は壁の生成スプライト名。TileNameWall のプランを実体化するときに使う
	TileNameDWall = "dwall"
	// TileNameFloor は床タイル名
	TileNameFloor = "floor"
	// TileNameDirt は土のタイル名
	TileNameDirt = "dirt"
	// TileNameVoid は虚空タイル名
	TileNameVoid = "void"
	// TileNameBridgeA は橋タイル名の A バリアント
	TileNameBridgeA = "bridge_a"
	// TileNameBridgeB は橋タイル名の B バリアント
	TileNameBridgeB = "bridge_b"
	// TileNameBridgeC は橋タイル名の C バリアント
	TileNameBridgeC = "bridge_c"
	// TileNameBridgeD は橋タイル名の D バリアント
	TileNameBridgeD = "bridge_d"
)
