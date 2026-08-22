package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dangerBoundaryRaws は危険度境界の検証用に、低層・単一危険度・高層の3エントリを持つ敵テーブルと
// アイテムテーブルを1つずつ持つ raws を作る。MinDanger/MaxDanger の一致と範囲外の判定を厳密に見る。
func dangerBoundaryRaws() *oapi.Raws {
	enemyTables := []oapi.EnemyTable{{
		Id:   "danger_enemies",
		Name: "danger_enemies",
		Entries: []oapi.EnemyTableEntry{
			{Id: "low", MinDanger: 1, MaxDanger: 3, Pack: "1d1", Weight: 1},
			{Id: "mid", MinDanger: 5, MaxDanger: 5, Pack: "1d1", Weight: 1},
			{Id: "high", MinDanger: 8, MaxDanger: 10, Pack: "1d1", Weight: 1},
		},
	}}
	itemTables := []oapi.ItemTable{{
		Id:   "danger_items",
		Name: "danger_items",
		Entries: []oapi.ItemTableEntry{
			{Id: "low", MinDanger: 1, MaxDanger: 3, Weight: 1},
			{Id: "mid", MinDanger: 5, MaxDanger: 5, Weight: 1},
			{Id: "high", MinDanger: 8, MaxDanger: 10, Weight: 1},
		},
	}}
	return &oapi.Raws{EnemyTables: &enemyTables, ItemTables: &itemTables}
}

var dangerBoundaryCases = []struct {
	name   string
	danger int
	want   []string
}{
	{"MinDanger一致は含む", 1, []string{"low"}},
	{"MaxDanger一致は含む", 3, []string{"low"}},
	{"単一危険度の境界に一致", 5, []string{"mid"}},
	{"範囲の谷間は空", 4, []string{}},
	{"MinDanger一致は含む_高層", 8, []string{"high"}},
	{"MaxDanger一致は含む_高層", 10, []string{"high"}},
	{"上限超過は空", 11, []string{}},
	{"下限未満は空", 0, []string{}},
}

func TestResolveItemSources_危険度境界でフィルタする(t *testing.T) {
	t.Parallel()
	raws := dangerBoundaryRaws()

	for _, tt := range dangerBoundaryCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveItemSources(raws, "danger_items", tt.danger)
			require.NoError(t, err)
			ids := make([]string, 0, len(got))
			for _, s := range got {
				ids = append(ids, s.GroupID)
			}
			assert.Equal(t, tt.want, ids)
		})
	}
}

func TestResolveEnemyEntries_危険度境界でフィルタする(t *testing.T) {
	t.Parallel()
	raws := dangerBoundaryRaws()

	for _, tt := range dangerBoundaryCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveEnemyEntries(raws, "danger_enemies", tt.danger)
			require.NoError(t, err)
			names := make([]string, 0, len(got))
			for _, s := range got {
				names = append(names, s.Name)
			}
			assert.Equal(t, tt.want, names)
		})
	}
}
