package balance

import (
	"regexp"
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
	assert.Contains(t, md, "## 戦闘リスク")

	// 敵テーブルは id 昇順で並ぶ。cave < forest < ruins_area
	iCave := strings.Index(md, "(cave)")
	iForest := strings.Index(md, "(forest)")
	iRuins := strings.Index(md, "(ruins_area)")
	require.Positive(t, iCave)
	require.Positive(t, iForest)
	require.Positive(t, iRuins)
	assert.Less(t, iCave, iForest, "cave が forest より前")
	assert.Less(t, iForest, iRuins, "forest が ruins_area より前")

	// 各テーブルに day1 行が出る。難易度カーブの行は 戦力比・目標の2列がともに小数で 内/外 で終わり、
	// 3列目が整数の経済や % を持つリスクの進行表と構造で区別できる。特定の戦力比の値に依存させないため、
	// 両小数+内外で数える。表は列幅で桁揃えされるので空白数に依らず正規表現で数える。
	day1Curve := regexp.MustCompile(`\|\s+1 \|\s+1 \|\s+\d+\.\d+ \|\s+\d+\.\d+ \|\s+[内外]`)
	assert.Len(t, day1Curve.FindAllString(md, -1), 5, "全5テーブルの day1 難易度カーブ行")
}
