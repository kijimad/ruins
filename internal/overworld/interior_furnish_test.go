package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteriorPropRaw_全施設の家具refが写像を持つ は、施設 content が生む家具が in-game で無言に欠落
// しないことを守る。各施設種別を Furnish し、KindFurniture の Ref がすべて 写像 PropRawName にあることを
// 確かめる。content へ家具を足して写像を忘れると、ここで落ちて気付ける。
func TestInteriorPropRaw_全施設の家具refが写像を持つ(t *testing.T) {
	t.Parallel()

	// 単室 Furnish と多部屋 FurnishBuilding の両経路をなめる。多部屋は民家の水回りなど別の家具を出すので
	// 両方を検査しないと写像漏れを見逃す
	small := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	big := interior.Rect{X: 0, Y: 0, W: 28, H: 20}
	door := interior.Vec{X: 10, Y: 13}
	bigDoor := interior.Vec{X: 14, Y: 0}
	check := func(fac string, placed []interior.Placed) {
		for _, p := range placed {
			if p.Kind != interior.KindFurniture {
				continue
			}
			_, ok := interior.PropRawName(p.Ref)
			assert.Truef(t, ok, "施設 %q の家具 %q は 写像を持つ", fac, p.Ref)
		}
	}
	for _, fac := range []string{"house", "store", "clinic", "office", "depot", "antique", "lab", ""} {
		check(fac, interior.Furnish(1, small, door, fac))
		_, placed := interior.FurnishBuilding(1, big, bigDoor, fac)
		check(fac, placed)
	}
}

// TestInteriorPropRaw_写像先のrawが実在する は、抽象 Ref の写像先がゲームの raw prop に存在することを
// 守る。raw 名の typo や raw 側の削除で spawn 時にエラーになる退行を、生成を待たず表の検査で捕まえる。
func TestInteriorPropRaw_写像先のrawが実在する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	for ref, name := range interior.PropRaws() {
		_, err := raw.GetProp(world.Resources.RawMaster, name)
		require.NoErrorf(t, err, "Ref %q の写像先 raw %q が実在する", ref, name)
	}
}
