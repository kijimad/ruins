package hud

import (
	"image/color"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/overworld"
)

// Data はすべてのHUDウィジェットが必要とするデータを統合する
type Data struct {
	GameInfo         GameInfoData
	MacroMap         MacroMapData
	DebugOverlay     DebugOverlayData
	MessageData      MessageData
	CurrencyData     CurrencyData
	WeaponSlotsData  WeaponSlotsData
	StatusBadgesData StatusBadgesData
}

// GameInfoData はゲーム基本情報のデータ
type GameInfoData struct {
	FloorNumber         int              // フロア番号
	PlayerHP            int              // プレイヤーの現在HP
	PlayerMaxHP         int              // プレイヤーの最大HP
	PlayerWeight        consts.Milligram // プレイヤーの現在所持重量
	PlayerMaxWeight     consts.Milligram // プレイヤーの所持可能重量
	TempArrow           TemperatureArrow // 体温変化の矢印。HP バーの左に出す
	BodyTempRatio       float64          // 体温ゲージの割合。0..1 で 0.5 が平熱
	BodyTempVisible     bool             // 体温ゲージを表示するか
	AmbientTemp         int              // プレイヤー位置の周囲気温℃
	AmbientTempVisible  bool             // 周囲気温を表示するか
	AmbientTempColor    color.RGBA       // 周囲気温の文字色。快適帯の内外を示す
	AmbientShelterLabel string           // プレイヤー位置の囲われの訳済み表示名。屋外・屋内・半屋外
	MessageAreaHeight   int              // メッセージエリアの高さ（ステータス表示位置計算用）
	ScreenDimensions    ScreenDimensions // 画面サイズ。階層表示位置計算用
}

// MacroMapData は右上のマクロ地図ウィジェットの描画データ。N キーで開く地形俯瞰と同じ内容を
// 縮小して常時表示する。
type MacroMapData struct {
	HasBand bool                // オーバーワールドにいて帯があるか。偽なら No Data を出す
	View    overworld.MacroView // 帯全体のチャンク俯瞰
	Config  MacroMapConfig      // パネル寸法と glyph 表示の閾値
	Screen  ScreenDimensions    // 右上配置に使う画面サイズ
}

// MacroMapConfig はマクロ地図ウィジェットの寸法。
type MacroMapConfig struct {
	Width, Height int // パネルのピクセル寸法
	MinGlyphPx    int // セル辺がこのpx以上のときだけ glyph を描く。下回れば地形色セルのみで見せる
}

// ScreenDimensions は画面サイズ
type ScreenDimensions struct {
	Width  int
	Height int
}

// DebugOverlayData はAIデバッグ情報のデータ
type DebugOverlayData struct {
	Enabled          bool              // デバッグ表示有効フラグ
	AIStates         []AIStateInfo     // AI状態情報
	VisionRanges     []VisionRangeInfo // 視界範囲情報
	HPDisplays       []HPDisplayInfo   // HP表示情報
	ScreenDimensions ScreenDimensions  // 画面サイズ
}

// AIStateInfo はAI状態の情報
type AIStateInfo struct {
	Screen    consts.Coord[consts.ScreenPixel] // 画面上の座標
	StateText string                           // 状態テキスト
}

// VisionRangeInfo は視界範囲の情報
type VisionRangeInfo struct {
	Screen       consts.Coord[consts.ScreenPixel] // 中心の画面座標
	ScaledRadius float32                          // スケール済み半径
}

// HPDisplayInfo はHP表示の情報
type HPDisplayInfo struct {
	Screen     consts.Coord[consts.ScreenPixel] // 画面上の座標
	CurrentHP  int                              // 現在のHP
	MaxHP      int                              // 最大HP
	EntityName string                           // エンティティ名（デバッグ用）
}

// MessageData はメッセージ表示に必要なデータ
type MessageData struct {
	Messages         []string          // 表示するメッセージ一覧
	ScreenDimensions ScreenDimensions  // 画面サイズ
	Config           MessageAreaConfig // メッセージエリア設定
}

// CurrencyData は通貨表示に必要なデータ
type CurrencyData struct {
	Currency         consts.Currency   // プレイヤーの所持地髄
	ScreenDimensions ScreenDimensions  // 画面サイズ
	Config           MessageAreaConfig // 位置計算にメッセージエリアの情報が必要
}

// WeaponSlotsData は武器スロット表示に必要なデータ
type WeaponSlotsData struct {
	Slots            []WeaponSlotInfo // 5つの武器スロット情報
	SelectedSlot     int              // 選択中のスロット番号（0-4）
	ScreenDimensions ScreenDimensions // 画面サイズ
}

// WeaponSlotInfo は武器スロットの情報
type WeaponSlotInfo struct {
	SlotNumber  gc.EquipmentSlotNumber // スロット番号
	WeaponName  string                 // 武器名（空文字列なら武器なし）
	SpriteSheet string                 // スプライトシート名
	SpriteName  string                 // スプライト名
}
