package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// TestUrbanEnemyTableFor_施設で敵テーブルを引き未登録は既定へ落ちる は、施設種別から敵テーブル名を
// 引く写像と、未登録施設が既定の廃墟テーブルへ落ちるフォールバックを固定する。
func TestUrbanEnemyTableFor_施設で敵テーブルを引き未登録は既定へ落ちる(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "downtown_enemies", urbanEnemyTableFor(facilityClinic), "診療所は高危険施設テーブル")
	assert.Equal(t, "downtown_enemies", urbanEnemyTableFor(facilityLab), "研究施設は診療所と同じ")
	assert.Equal(t, "industrial_enemies", urbanEnemyTableFor(facilityDepot), "倉庫は産業機械テーブル")
	assert.Equal(t, "industrial_enemies", urbanEnemyTableFor(facilityOffice), "事務所は倉庫と同じ")
	assert.Equal(t, urbanEnemyTable, urbanEnemyTableFor(facilityHouse), "未登録の住宅は既定の廃墟テーブル")
	assert.Equal(t, urbanEnemyTable, urbanEnemyTableFor(facilityType("unknown")), "未知の施設も既定へ落ちる")
}

// TestFacilityEnemyTable_全テーブル名が実在する は、写像の値と既定テーブルが raw に存在し
// GetEnemyTable が error にならないことを固定する。テーブル名の typo が生成時の runtime エラーに
// なるのを、この単体テストで前もって止める。
func TestFacilityEnemyTable_全テーブル名が実在する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	names := make([]string, 0, 1+len(facilityEnemyTable))
	names = append(names, urbanEnemyTable)
	for _, name := range facilityEnemyTable {
		names = append(names, name)
	}
	for _, name := range names {
		_, err := raw.GetEnemyTable(world.Resources.RawMaster, name)
		assert.NoErrorf(t, err, "敵テーブル %q が raw に存在する", name)
	}
}
