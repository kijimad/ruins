package balance

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetPowerRatio_境界と補間(t *testing.T) {
	t.Parallel()
	// day1 とそれ以前は下限クランプで 2.5、day20 とそれ以降は上限クランプで 1.3。
	assert.InDelta(t, 2.5, TargetPowerRatio(1), 1e-9, "day1 は始点")
	assert.InDelta(t, 2.5, TargetPowerRatio(0), 1e-9, "day1 未満はクランプ")
	assert.InDelta(t, 1.3, TargetPowerRatio(20), 1e-9, "day20 は終点")
	assert.InDelta(t, 1.3, TargetPowerRatio(21), 1e-9, "day20 超えはクランプ")
	// 中間は線形補間。day10 は始点と終点の間に入る
	mid := TargetPowerRatio(10)
	assert.Less(t, mid, 2.5)
	assert.Greater(t, mid, 1.3)
}

func TestBaseline_行数と範囲判定(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	rows, err := Baseline(master, BaselinePlayer, BaselineWeapon, BaselineAreaTable, BaselineDays)
	require.NoError(t, err)
	require.Len(t, rows, BaselineDays)

	for _, r := range rows {
		// InRange は目標帯 中心±targetBand の内外と一致する
		want := r.PowerRatio >= r.Target-targetBand && r.PowerRatio <= r.Target+targetBand
		assert.Equal(t, want, r.InRange, "day %d の範囲判定", r.Day)
		assert.InDelta(t, TargetPowerRatio(r.Day), r.Target, 1e-9, "day %d の目標値", r.Day)
	}
}

func TestRenderBaselineMarkdown_全区画とテーブル順を含む(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	md, err := RenderBaselineMarkdown(master, "ash", "bare_hands", 21)
	require.NoError(t, err)

	// 見出しと各セクションが出る
	assert.Contains(t, md, "# バランスベースライン")
	assert.Contains(t, md, "## 生存圧")
	assert.Contains(t, md, "## 感度")

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
