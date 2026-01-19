package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	w "github.com/kijimaD/ruins/internal/world"
)

// HUDRenderingSystem はクエリを実行し、UIを描画するシステム
type HUDRenderingSystem struct {
	gameInfo        *hud.GameInfo
	minimap         *hud.Minimap
	debugOverlay    *hud.DebugOverlay
	messageArea     *hud.MessageArea
	currencyDisplay *hud.CurrencyDisplay
	weaponSlots     *hud.WeaponSlots
	enabled         bool
}

// NewHUDRenderingSystem は新しいHUD描画システムを作成する
func NewHUDRenderingSystem(world w.World) *HUDRenderingSystem {
	hudFace := (*world.Resources.Faces)["dougenzaka"]
	defaultFace := world.Resources.UIResources.Text.Face
	bigTitleFace := world.Resources.UIResources.Text.BigTitleFace

	return &HUDRenderingSystem{
		gameInfo:        hud.NewGameInfo(hudFace, bigTitleFace),
		minimap:         hud.NewMinimap(defaultFace),
		debugOverlay:    hud.NewDebugOverlay(defaultFace),
		messageArea:     hud.NewMessageArea(world),
		currencyDisplay: hud.NewCurrencyDisplay(hudFace),
		weaponSlots:     hud.NewWeaponSlots(defaultFace),
		enabled:         true,
	}
}

// String はシステム名を返す
// w.Updater と w.Renderer のインターフェースを実装する
func (sys HUDRenderingSystem) String() string {
	return "HUDRenderingSystem"
}

// Draw はHUD描画を行う
// w.Renderer interfaceを実装
func (sys *HUDRenderingSystem) Draw(world w.World, screen *ebiten.Image) error {
	sys.Run(world, screen)
	return nil
}

// Update はHUDシステム全体を更新する
// w.Updater interfaceを実装
func (sys *HUDRenderingSystem) Update(world w.World) error {
	if sys == nil || !sys.enabled {
		return nil
	}

	// 各ウィジェットの更新処理
	if sys.gameInfo != nil {
		sys.gameInfo.Update(world)
	}
	if sys.minimap != nil {
		sys.minimap.Update(world)
	}
	if sys.debugOverlay != nil {
		sys.debugOverlay.Update(world)
	}
	if sys.messageArea != nil {
		sys.messageArea.Update(world)
	}
	if sys.currencyDisplay != nil {
		sys.currencyDisplay.Update(world)
	}
	return nil
}

// Run はHUD描画システムのメイン処理を実行する
// worldからデータを抽出し、HUDを描画する
func (sys *HUDRenderingSystem) Run(world w.World, screen *ebiten.Image) {
	if sys == nil || !sys.enabled {
		return
	}

	// worldから全HUDデータを一括抽出
	hudData := ExtractHUDData(world)

	// 各ウィジェットにデータを渡して描画する。描画順がある
	if sys.minimap != nil {
		sys.minimap.Draw(screen, hudData.MinimapData)
	}
	if sys.debugOverlay != nil {
		sys.debugOverlay.Draw(screen, hudData.DebugOverlay)
	}
	if sys.messageArea != nil {
		sys.messageArea.Draw(screen, hudData.MessageData)
	}
	if sys.currencyDisplay != nil {
		sys.currencyDisplay.Draw(screen, hudData.CurrencyData)
	}
	if sys.weaponSlots != nil {
		sys.weaponSlots.Draw(screen, hudData.WeaponSlotsData, world)
	}
	if sys.gameInfo != nil {
		sys.gameInfo.Draw(screen, hudData.GameInfo)
	}
}
