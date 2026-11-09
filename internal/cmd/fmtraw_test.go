package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatRaw(t *testing.T) {
	t.Parallel()

	src := `[[dropTables]]
name = "コウモリ"
id = "bat"
nothing = 0.9
[[dropTables.entries]]
weight = 1.0
id = "scrap_iron"
`
	got, err := formatRaw([]byte(src))
	require.NoError(t, err)
	assert.Equal(t, `[[dropTables]]
  id = "bat"
  name = "コウモリ"
  nothing = 0.9

  [[dropTables.entries]]
    id = "scrap_iron"
    weight = 1.0
`, string(got))
}

func TestFormatRaw_整形済みはそのまま(t *testing.T) {
	t.Parallel()

	src := `[[items]]
name = "a"
id = "a"
[items.melee]
damage = 1
`
	once, err := formatRaw([]byte(src))
	require.NoError(t, err)
	twice, err := formatRaw(once)
	require.NoError(t, err)
	assert.Equal(t, string(once), string(twice))
}

func TestFormatRaw_不正なTOMLはエラー(t *testing.T) {
	t.Parallel()

	_, err := formatRaw([]byte("id = "))
	require.Error(t, err)
}
