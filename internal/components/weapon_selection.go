package components

// WeaponSelection はプレイヤーが選択中の武器スロットを保持するシングルトン。
// 現在地と無関係な戦闘 UI 状態を独立して持つ。
type WeaponSelection struct {
	// Slot は選択中の武器スロット番号。1から5
	Slot int
}
