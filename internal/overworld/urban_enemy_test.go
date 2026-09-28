package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUrbanEnemyTableFor_割り当ては専用未割り当ては既定へ落ちる は挙動を固定する。割り当てのある施設は
// 既定でなく専用テーブルを引き、未知施設は既定へ落ちる。特定のテーブル id は綴りを追うだけの死んだ検査に
// なるので固定しない。id と割り当ての実在は別テストが担保する。
func TestUrbanEnemyTableFor_割り当ては専用未割り当ては既定へ落ちる(t *testing.T) {
	t.Parallel()

	master := testutil.InitTestWorld(t).Resources.RawMaster

	assigned, err := urbanEnemyTableFor(master, facilityClinic)
	require.NoError(t, err)
	assert.NotEqual(t, urbanEnemyTable, assigned.Id, "割り当てのある施設は既定でなく専用テーブルを引く")

	fallback, err := urbanEnemyTableFor(master, facilityType("unknown"))
	require.NoError(t, err)
	assert.Equal(t, urbanEnemyTable, fallback.Id, "未知の施設は既定へ落ちる")
}

// TestUrbanEnemyTableFor_割り当て先が実在しなければerror は、割り当て先が raw に無いとき silent に既定へ
// すり替えず error を返すことを固定する。
func TestUrbanEnemyTableFor_割り当て先が実在しなければerror(t *testing.T) {
	t.Parallel()

	master := testutil.InitTestWorld(t).Resources.RawMaster
	master.FacilityEnemyTables = &[]oapi.FacilityEnemyTable{
		{Facility: oapi.FacilityKind(facilityClinic), EnemyTable: "no_such_table"},
	}

	_, err := urbanEnemyTableFor(master, facilityClinic)
	require.Error(t, err, "割り当て先が実在しなければ設定ミスとして error")
}

// TestFacilityEnemyTableName_全施設種別を網羅する は、全 facilityType に割り当てがあることを固定し、施設を
// 足して割り当てを忘れる漏れを止める。all は facilityType の全定数と揃える。
func TestFacilityEnemyTableName_全施設種別を網羅する(t *testing.T) {
	t.Parallel()

	master := testutil.InitTestWorld(t).Resources.RawMaster
	all := []facilityType{
		facilityHouse, facilityStore, facilityOffice, facilityDepot,
		facilityAntique, facilityClinic, facilityLab,
	}
	for _, fac := range all {
		_, ok := raw.FacilityEnemyTableName(master, string(fac))
		assert.Truef(t, ok, "施設 %q に敵テーブルの割り当てがある", fac)
	}
}
