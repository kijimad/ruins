package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUrbanEnemyTableFor_施設で敵テーブルを引き未割り当ては既定へ落ちる は施設→敵テーブルの割り当てと
// 未割り当てのフォールバックを固定する。
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
	assert.Equal(t, "house_enemies", tableID(facilityHouse), "住宅は住宅テーブル")
	assert.Equal(t, "store_enemies", tableID(facilityStore), "商店は商店テーブル")
	assert.Equal(t, "antique_enemies", tableID(facilityAntique), "骨董品店は骨董品店テーブル")
	assert.Equal(t, urbanEnemyTable, tableID(facilityType("unknown")), "未知の施設は既定へ落ちる")
}

// TestFacilityEnemyTables_全施設種別を網羅する は、全 facilityType に割り当てがあることを固定し、施設を
// 足して割り当てを忘れる漏れを止める。all は facilityType の全定数と揃える。
func TestFacilityEnemyTables_全施設種別を網羅する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	all := []facilityType{
		facilityHouse, facilityStore, facilityOffice, facilityDepot,
		facilityAntique, facilityClinic, facilityLab,
	}
	for _, fac := range all {
		_, ok := raw.FacilityEnemyTableName(world.Resources.RawMaster, string(fac))
		assert.Truef(t, ok, "施設 %q に敵テーブルの割り当てがある", fac)
	}
}

// TestUrbanEnemyTableFor_割り当て先が実在しなければerror は、割り当て先が raw に無いとき silent に既定へ
// すり替えず error を返すことを固定する。
func TestUrbanEnemyTableFor_割り当て先が実在しなければerror(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	bad := []oapi.FacilityEnemyTable{{Facility: string(facilityClinic), EnemyTable: "no_such_table"}}
	world.Resources.RawMaster.FacilityEnemyTables = &bad

	_, err := urbanEnemyTableFor(world, facilityClinic)
	require.Error(t, err, "割り当て先が実在しなければ設定ミスとして error")
}

// TestFacilityEnemyTables_割り当て先と既定が実在する は、割り当て先と既定テーブルが実在することを固定し、
// テーブル名の typo を生成前に止める。
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
