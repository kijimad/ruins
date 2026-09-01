package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRemedy_最も重い一致不調を治療する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	hs := world.Components.HealthStatus.Get(player)
	hs.Parts[gc.BodyPartArms].SetCondition(gc.HealthCondition{Type: gc.ConditionFracture, Timer: 40, Severity: gc.TimerToSeverity(40)}) // 軽度
	hs.Parts[gc.BodyPartLegs].SetCondition(gc.HealthCondition{Type: gc.ConditionFracture, Timer: 80, Severity: gc.TimerToSeverity(80)}) // 重度
	hs.Parts[gc.BodyPartArms].SetCondition(gc.HealthCondition{Type: gc.ConditionLaceration, Timer: 60, Severity: gc.TimerToSeverity(60)})

	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "Splint"})
	remedy := &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionFracture}, Potency: 150}

	u := &UseItemBehavior{}
	treated := u.applyRemedy(player, world, remedy, item)

	assert.True(t, treated)
	// 最も重い骨折、脚の重度が治療される
	leg := hs.Parts[gc.BodyPartLegs].GetCondition(gc.ConditionFracture)
	require.NotNil(t, leg)
	assert.Equal(t, consts.Percent(150), leg.TendQuality, "最も重い骨折が治療済み")
	// 軽い骨折と切り傷は未治療のまま。アイテム1つで1不調だけ
	arm := hs.Parts[gc.BodyPartArms].GetCondition(gc.ConditionFracture)
	require.NotNil(t, arm)
	assert.Equal(t, consts.Percent(0), arm.TendQuality, "治療は最も重い1つだけ")
	lac := hs.Parts[gc.BodyPartArms].GetCondition(gc.ConditionLaceration)
	require.NotNil(t, lac)
	assert.Equal(t, consts.Percent(0), lac.TendQuality, "Treats 外は治療しない")
}

func TestApplyRemedy_一致しなければ何もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	hs := world.Components.HealthStatus.Get(player)
	hs.Parts[gc.BodyPartArms].SetCondition(gc.HealthCondition{Type: gc.ConditionLaceration, Timer: 60, Severity: gc.TimerToSeverity(60)})

	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "Splint"})
	remedy := &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionFracture}, Potency: 150}

	u := &UseItemBehavior{}
	treated := u.applyRemedy(player, world, remedy, item)

	assert.False(t, treated, "一致する不調が無ければ治療しない")
	lac := hs.Parts[gc.BodyPartArms].GetCondition(gc.ConditionLaceration)
	require.NotNil(t, lac)
	assert.Equal(t, consts.Percent(0), lac.TendQuality)
}

func TestIsRemedyOnly(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	u := &UseItemBehavior{}

	pure := world.ECS.NewEntity()
	world.Components.Remedy.Add(pure, &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionFracture}, Potency: 100})
	assert.True(t, u.isRemedyOnly(world, pure), "治療だけなら治療専用")

	combo := world.ECS.NewEntity()
	world.Components.Remedy.Add(combo, &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionFracture}, Potency: 100})
	world.Components.ProvidesHealing.Add(combo, &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 10})
	assert.False(t, u.isRemedyOnly(world, combo), "回復も持つなら治療専用でない")
}
