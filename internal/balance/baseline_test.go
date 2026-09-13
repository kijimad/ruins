package balance

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderBaselineMarkdown_全区画とテーブル順を含む(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	md, err := RenderBaselineMarkdown(master, "ash", "bare_hands", 21)
	require.NoError(t, err)

	// 見出しと各セクションが出る
	assert.Contains(t, md, "# Balance baseline")
	assert.Contains(t, md, "## survival pressure")
	assert.Contains(t, md, "## sensitivity")

	// 敵テーブルは id 昇順で並ぶ。cave < forest < ruins_area
	iCave := strings.Index(md, "(cave)")
	iForest := strings.Index(md, "(forest)")
	iRuins := strings.Index(md, "(ruins_area)")
	require.Positive(t, iCave)
	require.Positive(t, iForest)
	require.Positive(t, iRuins)
	assert.Less(t, iCave, iForest, "cave が forest より前")
	assert.Less(t, iForest, iRuins, "forest が ruins_area より前")

	// 各テーブルに day 行が21日ぶん出る
	assert.Equal(t, 3, strings.Count(md, "| 1 | 1 |"), "各テーブルの day1 行")
}
