package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUrbanEnemyTableFor_施設で敵テーブルを引き未割り当ては既定へ落ちる は、raw.toml の
// facilityEnemyTables による施設→敵テーブルの割り当てと、未割り当て施設が既定の廃墟テーブルへ
// 落ちるフォールバックを固定する。
func TestUrbanEnemyTableFor_施設で敵テーブルを引き未割り当ては既定へ落ちる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	assert.Equal(t, "downtown_enemies", urbanEnemyTableFor(world, facilityClinic), "診療所は高危険施設テーブル")
	assert.Equal(t, "downtown_enemies", urbanEnemyTableFor(world, facilityLab), "研究施設は診療所と同じ")
	assert.Equal(t, "industrial_enemies", urbanEnemyTableFor(world, facilityDepot), "倉庫は産業機械テーブル")
	assert.Equal(t, "industrial_enemies", urbanEnemyTableFor(world, facilityOffice), "事務所は倉庫と同じ")
	assert.Equal(t, urbanEnemyTable, urbanEnemyTableFor(world, facilityHouse), "未割り当ての住宅は既定の廃墟テーブル")
	assert.Equal(t, urbanEnemyTable, urbanEnemyTableFor(world, facilityType("unknown")), "未知の施設も既定へ落ちる")
}

// TestFacilityEnemyTables_割り当て先と既定が実在する は、raw.toml の facilityEnemyTables が指す
// 敵テーブルと既定テーブルが実在し GetEnemyTable が error にならないことを固定する。テーブル名の
// typo が生成時の runtime エラーになるのを、この単体テストで前もって止める。
func TestFacilityEnemyTables_割り当て先と既定が実在する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	master := world.Resources.RawMaster

	_, err := raw.GetEnemyTable(master, urbanEnemyTable)
	require.NoErrorf(t, err, "既定の敵テーブル %q が raw に存在する", urbanEnemyTable)

	for _, fe := range raw.PtrSlice(master.FacilityEnemyTables) {
		_, err := raw.GetEnemyTable(master, fe.EnemyTable)
		assert.NoErrorf(t, err, "施設 %q の割り当て先 %q が raw に存在する", fe.Facility, fe.EnemyTable)
	}
}
