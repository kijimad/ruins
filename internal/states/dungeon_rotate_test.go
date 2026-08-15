package states

import (
	"math"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/stretchr/testify/assert"
)

func TestDungeon3D_rotate(t *testing.T) {
	t.Parallel()

	d := &dungeon3D{sys: gs.NewRender3DSystem()}

	d.rotate(1)
	assert.Equal(t, 1, d.orient)
	assert.InDelta(t, math.Pi/4, d.sys.Yaw, 1e-9)

	d.rotate(-1)
	assert.Equal(t, 0, d.orient)

	// 反時計回りは環で7へ回り込む
	d.rotate(-1)
	assert.Equal(t, 7, d.orient)
	assert.InDelta(t, 7*math.Pi/4, d.sys.Yaw, 1e-9)

	// 8回転で一巡して元へ戻る
	for range 8 {
		d.rotate(1)
	}
	assert.Equal(t, 7, d.orient)
}

func TestDungeon3D_moveDir(t *testing.T) {
	t.Parallel()

	// orient0: 既定カメラは北から見下ろす。画面奥は南、画面右は西へ対応する。
	// Right キーは画面の右すなわち世界の西へ動かす。左右が逆にならないことを固定する
	d := &dungeon3D{orient: 0}
	assert.Equal(t, gc.DirectionDown, d.moveDir(gc.DirectionUp))
	assert.Equal(t, gc.DirectionLeft, d.moveDir(gc.DirectionRight))
	assert.Equal(t, gc.DirectionRight, d.moveDir(gc.DirectionLeft))

	// orient2 は90度。Up は西へ回る
	d.orient = 2
	assert.Equal(t, gc.DirectionLeft, d.moveDir(gc.DirectionUp))
}

func TestDungeonState_moveDir_delegation(t *testing.T) {
	t.Parallel()

	// 3D無効なら world 固定のまま
	st := &DungeonState{}
	assert.Equal(t, gc.DirectionUp, st.moveDir(gc.DirectionUp))

	// 3D有効なら dungeon3D へ委譲する
	st.three.enabled = true
	assert.Equal(t, gc.DirectionDown, st.moveDir(gc.DirectionUp))
}
