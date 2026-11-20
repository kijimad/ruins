package components

import "fmt"

// WeaponSelection はプレイヤーが選択中の手持ちの欄を保持するシングルトン。
// 現在地と無関係な戦闘 UI 状態を独立して持つ。
type WeaponSelection struct {
	// Slot は選択中の手持ちの欄。SlotHand1 から SlotHand5
	Slot EquipmentSlotNumber
}

// Validate は選択中の手持ちの欄が SlotHand1 から SlotHand5 の間にあるかを検査する。ロード直後など信頼できない入力に使う
func (s WeaponSelection) Validate() error {
	if _, ok := s.Slot.HandIndex(); !ok {
		return fmt.Errorf("weapon selection slot is not a weapon slot: %d", s.Slot)
	}
	return nil
}
