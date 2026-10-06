package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 特定のテーブル id は綴りを追うだけになるので固定しない。
func TestUrbanEnemyTableFor_施設行の敵テーブルを引く(t *testing.T) {
	t.Parallel()

	master := testutil.InitTestWorld(t).Resources.RawMaster

	et, err := urbanEnemyTableFor(master, "clinic")
	require.NoError(t, err)
	assert.NotEmpty(t, et.Id, "施設行の enemyTable が指すテーブルを引く")

	_, err = urbanEnemyTableFor(master, "unknown_facility")
	require.Error(t, err, "未登録の施設は silent フォールバックせず error")
}

func TestUrbanEnemyTableFor_敵テーブルが実在しなければerror(t *testing.T) {
	t.Parallel()

	master := testutil.InitTestWorld(t).Resources.RawMaster
	master.Facilities = &[]oapi.Facility{
		{Id: "clinic", EnemyTable: "no_such_table", Planner: oapi.Clinic},
	}

	_, err := urbanEnemyTableFor(master, "clinic")
	require.Error(t, err, "enemyTable が実在しなければ設定ミスとして error")
}

func TestFacilities_全施設が敵テーブルを引ける(t *testing.T) {
	t.Parallel()

	master := testutil.InitTestWorld(t).Resources.RawMaster
	facilities := raw.PtrSlice(master.Facilities)
	require.NotEmpty(t, facilities, "施設が定義されている")
	for _, f := range facilities {
		_, err := urbanEnemyTableFor(master, f.Id)
		assert.NoErrorf(t, err, "施設 %q の enemyTable が引ける", f.Id)
	}
}
