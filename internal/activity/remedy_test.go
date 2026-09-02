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

func TestApplyRemedy_積み重なった同種の傷から最も重い1つを治療する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	arm := &world.Components.HealthStatus.Get(player).Parts[gc.BodyPartArms]
	// 同じ腕に切り傷を独立に3つ積み、重症度をずらす。時間経過で Timer が分かれた状況を模す
	arm.AddCondition(gc.HealthCondition{Type: gc.ConditionLaceration, Timer: 30, Severity: gc.TimerToSeverity(30)}) // 軽度
	arm.AddCondition(gc.HealthCondition{Type: gc.ConditionLaceration, Timer: 80, Severity: gc.TimerToSeverity(80)}) // 重度
	arm.AddCondition(gc.HealthCondition{Type: gc.ConditionLaceration, Timer: 55, Severity: gc.TimerToSeverity(55)}) // 中度

	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "Bandage"})
	remedy := &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionLaceration}, Potency: 150}

	u := &UseItemBehavior{}
	require.True(t, u.applyRemedy(player, world, remedy, item))

	// 最も重い傷、Timer 80 の重度だけが治療される。時間経過で Timer が分かれても最重症を選ぶ
	tended := 0
	for i := range arm.Conditions {
		c := &arm.Conditions[i]
		if c.Timer == 80 {
			assert.Equal(t, consts.Percent(150), c.TendQuality, "最も重い傷が治療される")
		} else {
			assert.Equal(t, consts.Percent(0), c.TendQuality, "他の傷は未治療のまま")
		}
		if c.TendQuality > 0 {
			tended++
		}
	}
	assert.Equal(t, 1, tended, "治療されるのは1つだけ")
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

func TestUseItemBehavior_Validate_治療専用は治せる不調がないと使えない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 添え木は骨折を治す治療専用アイテム
	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "Splint"})
	world.Components.Consumable.Add(item, &gc.Consumable{})
	world.Components.Remedy.Add(item, &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionFracture}, Potency: 150})

	comp := &gc.Activity{BehaviorName: gc.BehaviorUseItem, Params: &gc.UseItemParams{Target: item}}
	u := &UseItemBehavior{}

	// 骨折が無ければ使う前に弾く
	var ve *UserError
	require.ErrorAs(t, u.Validate(comp, player, world), &ve, "治せる不調がなければ UserError で弾く")

	// 骨折があれば使える
	world.Components.HealthStatus.Get(player).Parts[gc.BodyPartLegs].SetCondition(
		gc.HealthCondition{Type: gc.ConditionFracture, Timer: 60, Severity: gc.TimerToSeverity(60)})
	assert.NoError(t, u.Validate(comp, player, world), "治せる不調があれば使える")
}

func TestUseItemBehavior_Validate_併用効果があれば空振りでも使える(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 治療に加えて回復も持つアイテムは、治せる不調が無くても回復で使える
	item := world.ECS.NewEntity()
	world.Components.Consumable.Add(item, &gc.Consumable{})
	world.Components.Remedy.Add(item, &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionFracture}, Potency: 150})
	world.Components.ProvidesHealing.Add(item, &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 10})

	comp := &gc.Activity{BehaviorName: gc.BehaviorUseItem, Params: &gc.UseItemParams{Target: item}}
	u := &UseItemBehavior{}
	assert.NoError(t, u.Validate(comp, player, world), "治療専用でなければ弾かない")
}
