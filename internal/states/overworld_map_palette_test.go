package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/stretchr/testify/assert"
)

// TestFacilityPalette_全施設記号に色がある は、地図の施設色が記号キーで揃っていることを固定する。
// 施設種別を追加して facilityPalette に色を足し忘れると既定の灰色へ落ちる退行を検知する。
// 記号キーで引く実装にしたので FacilityGlyphs() の並び順には依存しない。
func TestFacilityPalette_全施設記号に色がある(t *testing.T) {
	t.Parallel()

	for _, g := range overworld.FacilityGlyphs() {
		_, ok := facilityPalette[g.Label]
		assert.Truef(t, ok, "施設 %s(%c) に地図色が定義されている", g.Name, g.Label)
	}
}
