package interior

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadContents_GoレシピとrawTomlが一致する は、raw.toml へ移植した内装レシピ一式が Go の
// content_catalog / facility の関数とバイト等価であることを固定する。移植の取りこぼしをここで捕まえ、
// FurnishBuilding を raw 駆動へ切り替えたとき golden が動かないことを保証する。Go レシピを削除する段で
// この移行検査は golden に役目を譲って外す。
func TestLoadContents_GoレシピとrawTomlが一致する(t *testing.T) {
	t.Parallel()

	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	cs, err := LoadContents(master)
	require.NoError(t, err)

	facilities := []FacilityKind{facHouse, facStore, facAntique, facClinic, facLab, facOffice, facDepot}

	// content 一式を Go から組み直し、id ごとに loaded と突き合わせる
	wantContents := map[string]Content{}
	wantMain := map[FacilityKind][]string{}
	for _, f := range facilities {
		for _, c := range facilityVariants(f) {
			wantContents[c.ID] = c
			wantMain[f] = append(wantMain[f], c.ID)
		}
	}
	for _, c := range facilityVariants("__unknown__") {
		wantContents[c.ID] = c
	}
	for _, f := range facilities {
		for _, c := range roomCatalog(f) {
			wantContents[c.ID] = c
		}
		fb := backRoomContent(f)
		wantContents[fb.ID] = fb
	}

	for id, goC := range wantContents {
		got, ok := cs.byID[id]
		require.Truef(t, ok, "content %q が raw.toml に移植済み", id)
		assert.Equalf(t, goC, got, "content %q が Go と raw.toml で一致する", id)
	}
	assert.Len(t, cs.byID, len(wantContents), "raw.toml の content 数が Go と一致する")

	for f, ids := range wantMain {
		assert.Equalf(t, ids, cs.facilityMain[f], "施設 %q の主室変種", f)
	}
	for _, f := range facilities {
		got := cs.facilityRooms[f]
		for role, c := range roomCatalog(f) {
			assert.Equalf(t, c.ID, got.rooms[role], "施設 %q の役割 %q", f, role)
		}
		assert.Equalf(t, backRoomContent(f).ID, got.fallback, "施設 %q の奥室フォールバック", f)
	}
}
