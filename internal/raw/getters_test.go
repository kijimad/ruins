package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestGetFacility(t *testing.T) {
	t.Parallel()
	raws := oapi.Raws{Facilities: &[]oapi.Facility{{Id: "工房"}}}

	t.Run("存在するIDは施設を返す", func(t *testing.T) {
		t.Parallel()
		got, ok := GetFacility(raws, "工房")
		assert.True(t, ok)
		assert.Equal(t, "工房", got.Id)
	})

	t.Run("存在しないIDはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, ok := GetFacility(raws, "存在しない")
		assert.False(t, ok)
	})
}

func TestGetLandmark(t *testing.T) {
	t.Parallel()
	raws := oapi.Raws{Landmarks: &[]oapi.Landmark{{Id: "遺跡"}}}

	t.Run("存在するIDはランドマークを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := GetLandmark(raws, "遺跡")
		assert.True(t, ok)
		assert.Equal(t, "遺跡", got.Id)
	})

	t.Run("存在しないIDはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, ok := GetLandmark(raws, "存在しない")
		assert.False(t, ok)
	})
}

func TestGetScatterZone(t *testing.T) {
	t.Parallel()
	raws := oapi.Raws{ScatterZones: &[]oapi.ScatterZone{{Id: "草むら"}}}

	t.Run("存在するIDは散布ゾーンを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := GetScatterZone(raws, "草むら")
		assert.True(t, ok)
		assert.Equal(t, "草むら", got.Id)
	})

	t.Run("存在しないIDはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, ok := GetScatterZone(raws, "存在しない")
		assert.False(t, ok)
	})
}

func TestGetMapGlyph(t *testing.T) {
	t.Parallel()
	raws := oapi.Raws{MapGlyphs: &[]oapi.MapGlyph{{Id: "工房", Glyph: "工"}}}

	t.Run("存在するIDは地図記号を返す", func(t *testing.T) {
		t.Parallel()
		got, ok := GetMapGlyph(raws, "工房")
		assert.True(t, ok)
		assert.Equal(t, "工", got.Glyph)
	})

	t.Run("存在しないIDはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, ok := GetMapGlyph(raws, "存在しない")
		assert.False(t, ok)
	})
}
