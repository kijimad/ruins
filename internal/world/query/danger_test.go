package query

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestDangerLevel(t *testing.T) {
	t.Parallel()

	t.Run("距離が増えると危険度は下がらない", func(t *testing.T) {
		t.Parallel()
		near := DangerLevel(0, 0)
		far := DangerLevel(consts.AbsTileX(dangerTilesPerLevel*4), 0)
		assert.GreaterOrEqual(t, far, near)
		assert.Greater(t, far, near, "4段ぶん離れれば危険度は上がるはず")
	})

	t.Run("日数が増えると危険度は下がらない", func(t *testing.T) {
		t.Parallel()
		early := DangerLevel(0, 1)
		late := DangerLevel(0, dangerDaysPerLevel*4)
		assert.Greater(t, late, early, "日数が進めば危険度は上がるはず")
	})

	t.Run("距離が主で日数が従", func(t *testing.T) {
		t.Parallel()
		// 同じ段数の増分なら、距離ぶんが日数ぶんより効く配合であることを固定する
		byDist := DangerLevel(consts.AbsTileX(dangerTilesPerLevel), 0)
		byDays := DangerLevel(0, dangerDaysPerLevel)
		assert.Equal(t, byDist, byDays, "1段ぶんの距離と日数は同じ危険度を足す暫定配合")
	})

	t.Run("負の入力は0として扱う", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, DangerLevel(-100, -5))
	})

	t.Run("同じ入力は常に同じ値", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, DangerLevel(120, 7), DangerLevel(120, 7))
	})
}
