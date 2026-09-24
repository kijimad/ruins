package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
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
	tableID := func(fac facilityType) string {
		et, err := urbanEnemyTableFor(world, fac)
		require.NoError(t, err)
		return et.Id
	}

	assert.Equal(t, "clinic_enemies", tableID(facilityClinic), "診療所は診療所テーブル")
	assert.Equal(t, "lab_enemies", tableID(facilityLab), "研究施設は研究施設テーブル")
	assert.Equal(t, "depot_enemies", tableID(facilityDepot), "倉庫は倉庫テーブル")
	assert.Equal(t, "office_enemies", tableID(facilityOffice), "事務所は事務所テーブル")
	assert.Equal(t, urbanEnemyTable, tableID(facilityHouse), "未割り当ての住宅は既定の廃墟テーブル")
	assert.Equal(t, urbanEnemyTable, tableID(facilityType("unknown")), "未知の施設も既定へ落ちる")
}

// TestUrbanEnemyTableFor_割り当て先が実在しなければerror は、施設に割り当てた敵テーブルが raw に
// 無いとき silent に既定へすり替えず error を返すことを固定する。raw.toml の設定ミスを生成時に露見させる。
func TestUrbanEnemyTableFor_割り当て先が実在しなければerror(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	bad := []oapi.FacilityEnemyTable{{Facility: string(facilityClinic), EnemyTable: "no_such_table"}}
	world.Resources.RawMaster.FacilityEnemyTables = &bad

	_, err := urbanEnemyTableFor(world, facilityClinic)
	require.Error(t, err, "割り当て先が実在しなければ設定ミスとして error")
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
