package components

// WeaponSelection はプレイヤーが選択中の武器スロットを保持するシングルトン。
// 選択スロットはダンジョンや現在地と無関係な戦闘 UI 状態なので Dungeon から分離する。
type WeaponSelection struct {
	// Slot は選択中の武器スロット番号。1から5
	Slot int
}
