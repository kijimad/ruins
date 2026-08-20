package keybind

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNavHint はキー操作ヒントが束縛表から導出されることを固定する。
// 同じ Label の連続行はキー表記が連結され、Cancel の行は末尾へ回る
func TestNavHint(t *testing.T) {
	t.Parallel()

	detail := []Binding{
		{Key: ebiten.KeyX, Shift: ShiftForbidden, Action: inputmapper.ActionOpenItemDetail, Label: "Details"},
	}

	t.Run("表の順に並べ閉じる操作を末尾へ回す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		got := NavHint(world, MustMerge(MenuCommon, detail))

		want := consts.IconArrowLeft + consts.IconArrowRight + " Tab   " +
			consts.IconArrowUp + consts.IconArrowDown + " Select   " +
			consts.IconKeyEnter + " Confirm   " +
			consts.IconQuestion + " Help   " +
			string(consts.IconKeyAlphaBase+'x'-'a') + " Details   " +
			consts.IconKeyEsc + " Back"
		assert.Equal(t, want, got)
	})

	t.Run("Shift併用はShift記号を前置する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		got := NavHint(world, []Binding{
			{Key: ebiten.KeyX, Shift: ShiftRequired, Action: inputmapper.ActionVerbExamine, Label: "Inspect"},
		})

		assert.Equal(t, consts.IconKeyShift+string(consts.IconKeyAlphaBase+'x'-'a')+" Inspect", got)
	})
}

// TestHelpHint はフッター常設の入口ヒントがヘルプ行だけに絞られることを固定する
func TestHelpHint(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	assert.Equal(t, consts.IconQuestion+" Help", HelpHint(world))
}

// TestKeyLabel_キーキャップ表記 は ebiten の内部名が表示へ漏れないことを固定する。
// 数字キーの String は Digit1 のような内部名を返すため、グリフへの写像を挟まないと
// ヘルプの連結表記が digit1digit2 のように壊れる
func TestKeyLabel_キーキャップ表記(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	weaponSlots := []Binding{
		{Key: ebiten.Key1, Action: inputmapper.ActionSwitchWeaponSlot1, Label: "Weapon slot"},
		{Key: ebiten.Key2, Action: inputmapper.ActionSwitchWeaponSlot2, Label: "Weapon slot"},
		{Key: ebiten.Key3, Action: inputmapper.ActionSwitchWeaponSlot3, Label: "Weapon slot"},
		{Key: ebiten.Key4, Action: inputmapper.ActionSwitchWeaponSlot4, Label: "Weapon slot"},
		{Key: ebiten.Key5, Action: inputmapper.ActionSwitchWeaponSlot5, Label: "Weapon slot"},
	}
	entries := HintEntries(world, weaponSlots)
	require.Len(t, entries, 1, "同じラベルの連続行は1項目にまとまる")
	var wantDigits strings.Builder
	for n := rune(1); n <= 5; n++ {
		wantDigits.WriteString(string(consts.IconKeyDigitBase + n))
	}
	assert.Equal(t, wantDigits.String(), entries[0].Keys, "数字は数字グリフで連結される")

	assert.Equal(t, consts.IconKeyDot, KeyLabel(Binding{Key: ebiten.KeyPeriod}), "記号キーも記号グリフで表す")
	assert.Equal(t, consts.IconKeySpace, KeyLabel(Binding{Key: ebiten.KeySpace}), "Spaceはキーキャップグリフで表す")
}
