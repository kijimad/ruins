package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestLevel_Width(t *testing.T) {
	t.Parallel()

	l := &Level{TileWidth: 10, TileHeight: 5}
	assert.Equal(t, consts.Pixel(10*int(consts.TileSize)), l.Width())
}

func TestLevel_Height(t *testing.T) {
	t.Parallel()

	l := &Level{TileWidth: 10, TileHeight: 5}
	assert.Equal(t, consts.Pixel(5*int(consts.TileSize)), l.Height())
}
