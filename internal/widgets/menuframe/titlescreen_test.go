package menuframe

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

func TestTitleScreen_一覧の後に注記を積む順で並べる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	res := world.Resources.UIResources
	rows := []Row{
		{Cells: styled.TextCells("はじめから")},
		{Cells: styled.TextCells("設定")},
	}
	notes := []string{"v1.0.0", "Build 123"}

	root := TitleScreen(world, res, 0, rows, notes)

	assert.Equal(t, []string{"はじめから", "設定", "v1.0.0", "Build 123"}, uicore.CollectLabels(root))
}

func TestTitleScreen_注記が無ければ一覧だけになる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	res := world.Resources.UIResources
	rows := []Row{{Cells: styled.TextCells("はじめから")}}

	root := TitleScreen(world, res, 0, rows, nil)

	assert.Equal(t, []string{"はじめから"}, uicore.CollectLabels(root))
}
