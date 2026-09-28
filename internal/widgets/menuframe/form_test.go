package menuframe_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

func TestFormScreen_エラー文言が空ならエラー行を持たない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	res := world.Resources.UIResources
	body := uicore.NewText("Ash", res.Text.BodyFace, theme.TextPrimary)

	root := menuframe.FormScreen(world, res, "名前", body, "", "Enterで確定")

	assert.Equal(t, []string{"名前", "Ash", "Enterで確定"}, uicore.CollectLabels(root), "見出し・入力欄・ヒントの3つだけ")
}

func TestFormScreen_エラー文言があればヒントの手前に加わる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	res := world.Resources.UIResources
	body := uicore.NewText("Ash", res.Text.BodyFace, theme.TextPrimary)

	root := menuframe.FormScreen(world, res, "名前", body, "3文字以上で入力してください", "Enterで確定")

	assert.Equal(t, []string{"名前", "Ash", "3文字以上で入力してください", "Enterで確定"}, uicore.CollectLabels(root))
}
