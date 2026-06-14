package theme

import "image/color"

// ========== ヘルパー ==========

// rgb はRGB値からcolor.RGBAを作成する
func rgb(v uint64) color.RGBA {
	return color.RGBA{
		R: uint8((v >> 16) & 0xFF),
		G: uint8((v >> 8) & 0xFF),
		B: uint8(v & 0xFF),
		A: 0xFF,
	}
}

// ========== テキスト色 ==========

var (
	// TextPrimary は見出し・タイトルに使う白色
	TextPrimary = rgb(0xf5f5f5)
	// TextSecondary は補助テキストに使うグレー
	TextSecondary = rgb(0xa9a9a9)
	// TextAccent はアクセントカラーとして使う緑色
	TextAccent = rgb(0x9dd793)
	// TextSelected は選択中の項目に使う明るい白
	TextSelected = color.RGBA{R: 255, G: 255, B: 240, A: 255}
	// TextDisabled は無効状態のテキスト色
	TextDisabled = rgb(0x5a7a91)
)

// ========== カーソル色 ==========

var (
	// CursorPickup は拾うモードのカーソル色。青色
	CursorPickup = color.RGBA{R: 50, G: 200, B: 255, A: 255}
	// CursorPlace は置くモードのカーソル色。緑色
	CursorPlace = color.RGBA{R: 50, G: 255, B: 100, A: 255}
	// CursorShoot は射撃モードのカーソル色。赤色
	CursorShoot = color.RGBA{R: 255, G: 50, B: 50, A: 255}
	// CursorLook は調査モードのカーソル色。白色
	CursorLook = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

// ========== パネル色 ==========

var (
	// PanelBackground はパネルの背景色
	PanelBackground = color.RGBA{R: 12, G: 18, B: 30, A: 220}
	// PanelHighlight はパネル上辺のハイライト線
	PanelHighlight = color.RGBA{R: 60, G: 75, B: 100, A: 180}
	// PanelShadow はパネル下辺のシャドウ線
	PanelShadow = color.RGBA{R: 0, G: 0, B: 10, A: 200}
	// Overlay はモーダル背景の半透明黒
	Overlay = color.RGBA{R: 0, G: 0, B: 0, A: 200}
	// ScreenBackground は画面クリア用の背景色
	ScreenBackground = color.RGBA{R: 30, G: 30, B: 30, A: 255}
	// Transparent は完全透明色。ebitenuiの透明背景に使用する
	Transparent = color.NRGBA{}
)

// ========== UI要素色 ==========

var (
	// ListSelectedBg はリスト選択項目の背景色
	ListSelectedBg = rgb(0x4b687a)
	// ListFocusedBg はリストフォーカス項目の背景色
	ListFocusedBg = rgb(0x2a3944)
	// InputCaret はテキスト入力のカーソル色
	InputCaret = rgb(0xe7c34b)
)

// ========== メッセージウィンドウ色 ==========

var (
	// WindowBackground はメッセージウィンドウの背景色
	WindowBackground = color.RGBA{R: 20, G: 20, B: 30, A: 240}
	// WindowBorder はメッセージウィンドウの枠線色
	WindowBorder = color.RGBA{R: 100, G: 100, B: 120, A: 255}
	// WindowActionBg はアクション領域の背景色
	WindowActionBg = color.RGBA{R: 40, G: 40, B: 50, A: 255}
	// WindowActionText はアクション領域のテキスト色
	WindowActionText = color.RGBA{R: 180, G: 180, B: 200, A: 255}
	// ChoiceSelectedBg は選択肢の選択中背景色
	ChoiceSelectedBg = color.NRGBA{R: 75, G: 104, B: 122, A: 100}
)

// ========== HUD色 ==========

var (
	// HUDSlotSelectedBorder は選択中スロットの枠線色
	HUDSlotSelectedBorder = color.RGBA{R: 255, G: 255, B: 100, A: 255}
	// HUDSlotBorder は非選択スロットの枠線色
	HUDSlotBorder = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	// HUDBadgeBg はバッジの背景色
	HUDBadgeBg = color.RGBA{R: 100, G: 100, B: 100, A: 255}
	// HUDGaugeBg はHPゲージの背景色
	HUDGaugeBg = color.RGBA{R: 100, G: 0, B: 0, A: 255}
	// HUDGaugeBorder はHPゲージの白い枠線色
	HUDGaugeBorder = color.RGBA{R: 220, G: 220, B: 220, A: 230}
	// HUDMinimapBg はミニマップの背景色
	HUDMinimapBg = color.RGBA{R: 0, G: 0, B: 0, A: 128}
	// HUDPlayerMarker はミニマップのプレイヤー表示色
	HUDPlayerMarker = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	// HUDTextOutline はテキストのアウトライン色
	HUDTextOutline = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	// HUDWeightDanger は重量超過時のテキスト色
	HUDWeightDanger = color.RGBA{R: 255, G: 50, B: 50, A: 255}
	// HUDWeightWarning は重量警告時のテキスト色
	HUDWeightWarning = color.RGBA{R: 255, G: 200, B: 0, A: 255}
)

// ========== ステータス色 ==========

var (
	// StatusSuccess は成功色
	StatusSuccess = rgb(0x198754)
	// StatusDanger は危険色
	StatusDanger = rgb(0xdc3545)
)

// ========== 属性色 ==========

var (
	// ElementFire は炎属性色
	ElementFire = rgb(0xc44303)
	// ElementThunder は雷属性色
	ElementThunder = rgb(0x4169e1)
	// ElementChill は冷気属性色
	ElementChill = rgb(0x00ffff)
	// ElementPhoton は光子属性色
	ElementPhoton = rgb(0xffff00)
)
