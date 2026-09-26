package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUrbanEnemyTableFor_施設行の敵テーブルを引く は挙動を固定する。施設行の enemyTable が指すテーブルを
// 引き、未登録の施設は silent フォールバックせず error にする。特定のテーブル id は綴りを追うだけの死んだ
// 検査になるので固定しない。id の実在は別テストが担保する。
func TestUrbanEnemyTableFor_施設行の敵テーブルを引く(t *testing.T) {
	t.Parallel()

	master := testutil.InitTestWorld(t).Resources.RawMaster

	et, err := urbanEnemyTableFor(master, "clinic")
	require.NoError(t, err)
	assert.NotEmpty(t, et.Id, "施設行の enemyTable が指すテーブルを引く")

	_, err = urbanEnemyTableFor(master, "unknown_facility")
	require.Error(t, err, "未登録の施設は silent フォールバックせず error")
}

// TestUrbanEnemyTableFor_敵テーブルが実在しなければerror は、施設行の enemyTable が raw に無いとき silent に
// すり替えず error を返すことを固定する。
func TestUrbanEnemyTableFor_敵テーブルが実在しなければerror(t *testing.T) {
	t.Parallel()

	master := testutil.InitTestWorld(t).Resources.RawMaster
	master.Facilities = &[]oapi.Facility{
		{Id: "clinic", EnemyTable: "no_such_table", Planner: oapi.Clinic},
	}

	_, err := urbanEnemyTableFor(master, "clinic")
	require.Error(t, err, "enemyTable が実在しなければ設定ミスとして error")
}

// TestFacilities_全施設が敵テーブルを引ける は、全 facilities 行の enemyTable が実在テーブルを引けることを
// 固定し、施設を足して割り当てを忘れる漏れを止める。分母は raw の facilities 行そのもの。
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
