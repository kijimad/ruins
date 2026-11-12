package raw_test

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/require"
)

func TestSelectByWeight(t *testing.T) {
	t.Parallel()
	ct := rawschema.CommandTable{
		Name: "TEST",
		Entries: []rawschema.CommandTableEntry{
			{
				Id:     "A",
				Weight: 0.5,
			},
			{
				Id:     "B",
				Weight: 0.2,
			},
			{
				Id:     "C",
				Weight: 0.3,
			},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	_, err := raw.SelectCommandByWeight(ct, rng)
	require.NoError(t, err)
}
