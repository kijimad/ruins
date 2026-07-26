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
// しないことを守る。各施設種別を Furnish し、KindFurniture の Ref がすべて interiorPropRaw にあることを
// 確かめる。content へ家具を足して写像を忘れると、ここで落ちて気付ける。
func TestInteriorPropRaw_全施設の家具refが写像を持つ(t *testing.T) {
	t.Parallel()

	footprint := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	door := interior.Vec{X: 10, Y: 13}
	for _, fac := range []string{"house", "store", "clinic", "office", "depot", "antique", "lab", ""} {
		for _, p := range interior.Furnish(1, footprint, door, fac) {
			if p.Kind != interior.KindFurniture {
				continue
			}
			_, ok := interiorPropRaw[p.Ref]
			assert.Truef(t, ok, "施設 %q の家具 %q は interiorPropRaw に写像を持つ", fac, p.Ref)
		}
	}
}

// TestInteriorPropRaw_写像先のrawが実在する は、抽象 Ref の写像先がゲームの raw prop に存在することを
// 守る。raw 名の typo や raw 側の削除で spawn 時にエラーになる退行を、生成を待たず表の検査で捕まえる。
func TestInteriorPropRaw_写像先のrawが実在する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	for ref, name := range interiorPropRaw {
		_, err := raw.GetProp(world.Resources.RawMaster, name)
		require.NoErrorf(t, err, "Ref %q の写像先 raw %q が実在する", ref, name)
	}
}
