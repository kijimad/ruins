package rawcheck_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/rawschema/rawcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_実データが通る(t *testing.T) {
	t.Parallel()
	raws, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	require.NoError(t, rawcheck.Validate(raws))
}

func TestValidate_違反を位置で返す(t *testing.T) {
	t.Parallel()
	raws := rawschema.Raws{Items: &[]rawschema.Item{{Id: "x", Name: "x", Description: "d", SpriteKey: "Bad Key", SpriteSheetName: "s"}}}
	err := rawcheck.Validate(raws)
	var ce *rawcheck.Error
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, []string{"items.0.spriteKey"}, ce.Paths())
}

func TestValidate_ダイス表記の違反を位置で返す(t *testing.T) {
	t.Parallel()
	// 個数0の 0d6 は Dice の正規表現が弾く
	raws := makeRefRaws(func(r *rawschema.Raws) { (*r.EnemyTables)[0].Entries[0].Pack = "0d6" })
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.Validate(raws), &ce)
	assert.Equal(t, []string{"enemyTables.0.entries.0.pack"}, ce.Paths())
}
