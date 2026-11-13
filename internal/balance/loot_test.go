package balance

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateFacilityLoot_施設ごとに全体と各絵の分布を出す(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)

	const trials = 300
	got, err := GenerateFacilityLoot(master, trials, 1)
	require.NoError(t, err)

	facilities := raw.PtrSlice(master.Facilities)
	require.Len(t, got, len(facilities))
	for i, f := range got {
		assert.Equal(t, facilities[i].Id, f.Facility)
		require.NotEmpty(t, f.Templates, f.Facility)
		assert.Equal(t, facilityLootAll, f.Templates[0].Template, "全体を先頭に置く")
		assert.Equal(t, trials, f.Templates[0].Trials)
		assert.NotEmpty(t, f.Templates[0].Items, "%s の建物から戦利品が出る", f.Facility)

		names := []string{facilityLootAll}
		picked := 0
		for _, tl := range f.Templates[1:] {
			names = append(names, tl.Template)
			picked += tl.Trials
		}
		assert.ElementsMatch(t, append([]string{facilityLootAll}, loader.FacilityChunkNames(f.Facility)...), names)
		assert.Equal(t, trials, picked, "各絵が選ばれた試行数の和は試行数")
	}
}

func TestGenerateFacilityLoot_同じseedは同じ結果(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	want, err := GenerateFacilityLoot(master, 50, 7)
	require.NoError(t, err)
	got, err := GenerateFacilityLoot(master, 50, 7)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestResolveCellLoot_表が引けなければエラー(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	rng := rand.New(rand.NewPCG(1, 2))

	_, err := resolveCellLoot(master, maptemplate.MapCell{Prop: "closet", LootTable: "no_such_table"}, rng)
	var notFound raw.KeyNotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, raw.KeyNotFoundError{Key: "no_such_table", Collection: "ItemTables"}, notFound)
}
