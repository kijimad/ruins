package components

// WeaponSelection はプレイヤーが選択中の武器スロットを保持するシングルトン。
// 現在地と無関係な戦闘 UI 状態を独立して持つ。
type WeaponSelection struct {
	// Index は選択中の武器スロットの、武器の並びでの添字。0 から WeaponSlotCount-1
	Index int
}
