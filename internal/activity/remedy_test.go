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

func TestTreatableConditions(t *testing.T) {
	t.Parallel()

	hs := &gc.HealthStatus{}
	hs.Parts[gc.BodyPartArms].AddCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: 80, Severity: gc.TimerToSeverity(80), TendQuality: 100})
	hs.Parts[gc.BodyPartArms].AddCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: 30, Severity: gc.TimerToSeverity(30), TendQuality: 50})
	hs.Parts[gc.BodyPartArms].AddCondition(gc.HealthCondition{Type: gc.ConditionType("fracture"), Timer: 60, Severity: gc.TimerToSeverity(60)})
	hs.Parts[gc.BodyPartLegs].AddCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: 40, Severity: gc.TimerToSeverity(40)})

	tests := []struct {
		name    string
		potency consts.Percent
		want    []gc.ConditionRef
	}{
		{
			name:    "質の高い物は質の低い治療済みと未治療を並べる",
			potency: 100,
			want:    []gc.ConditionRef{{Part: gc.BodyPartArms, Index: 1}, {Part: gc.BodyPartLegs, Index: 0}},
		},
		{
			name:    "質の低い物は同じ質以上の治療済みを除く",
			potency: 50,
			want:    []gc.ConditionRef{{Part: gc.BodyPartLegs, Index: 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			remedy := &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionType("laceration")}, Potency: tt.potency}
			assert.Equal(t, tt.want, TreatableConditions(hs, remedy))
		})
	}
}

func TestApplyRemedy_選んだ不調だけを治療する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	arm := &world.Components.HealthStatus.Get(player).Parts[gc.BodyPartArms]
	arm.AddCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: 80, Severity: gc.TimerToSeverity(80)})
	arm.AddCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: 30, Severity: gc.TimerToSeverity(30)})

	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "Rag"})
	remedy := &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionType("laceration")}, Potency: 50}

	u := &UseItemBehavior{}
	require.True(t, u.applyRemedy(player, world, remedy, gc.ConditionRef{Part: gc.BodyPartArms, Index: 1}, item))

	got := make([]consts.Percent, 0, len(arm.Conditions))
	for _, c := range arm.Conditions {
		got = append(got, c.TendQuality)
	}
	assert.Equal(t, []consts.Percent{0, 50}, got)
}

func TestApplyRemedy_治療できない不調を指すと何もしない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  gc.ConditionRef
	}{
		{name: "治療済みで質が同じ", ref: gc.ConditionRef{Part: gc.BodyPartArms, Index: 0}},
		{name: "治せない種類", ref: gc.ConditionRef{Part: gc.BodyPartArms, Index: 1}},
		{name: "添字が範囲外", ref: gc.ConditionRef{Part: gc.BodyPartArms, Index: 2}},
		{name: "部位が範囲外", ref: gc.ConditionRef{Part: gc.BodyPartCount, Index: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			arm := &world.Components.HealthStatus.Get(player).Parts[gc.BodyPartArms]
			arm.AddCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: 60, Severity: gc.TimerToSeverity(60), TendQuality: 50})
			arm.AddCondition(gc.HealthCondition{Type: gc.ConditionType("fracture"), Timer: 60, Severity: gc.TimerToSeverity(60)})

			item := world.ECS.NewEntity()
			world.Components.Name.Add(item, &gc.Name{Name: "Rag"})
			remedy := &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionType("laceration")}, Potency: 50}

			u := &UseItemBehavior{}
			assert.False(t, u.applyRemedy(player, world, remedy, tt.ref, item))
			assert.Equal(t, consts.Percent(0), arm.Conditions[1].TendQuality)
		})
	}
}

func TestIsRemedyOnly(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	u := &UseItemBehavior{}

	pure := world.ECS.NewEntity()
	world.Components.Remedy.Add(pure, &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionType("fracture")}, Potency: 100})
	assert.True(t, u.isRemedyOnly(world, pure), "治療だけなら治療専用")

	combo := world.ECS.NewEntity()
	world.Components.Remedy.Add(combo, &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionType("fracture")}, Potency: 100})
	world.Components.ProvidesHealing.Add(combo, &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 10})
	assert.False(t, u.isRemedyOnly(world, combo), "回復も持つなら治療専用でない")
}

func TestUseItemBehavior_Validate_治療専用(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tend    *gc.ConditionRef
		wantErr bool
	}{
		{name: "治療できる不調を選べば使える", tend: &gc.ConditionRef{Part: gc.BodyPartLegs, Index: 0}},
		{name: "不調を選ばないと使えない", tend: nil, wantErr: true},
		{name: "治療済みで質の上がらない不調を選ぶと使えない", tend: &gc.ConditionRef{Part: gc.BodyPartArms, Index: 0}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			hs := world.Components.HealthStatus.Get(player)
			hs.Parts[gc.BodyPartArms].SetCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: 60, Severity: gc.TimerToSeverity(60), TendQuality: 100})
			hs.Parts[gc.BodyPartLegs].SetCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: 60, Severity: gc.TimerToSeverity(60)})

			item := world.ECS.NewEntity()
			world.Components.Name.Add(item, &gc.Name{Name: "Bandage"})
			world.Components.Consumable.Add(item, &gc.Consumable{})
			world.Components.Remedy.Add(item, &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionType("laceration")}, Potency: 100})

			comp := NewUseItemActivity(item, tt.tend)
			u := &UseItemBehavior{}
			err = u.Validate(comp, player, world)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			var ve *UserError
			require.ErrorAs(t, err, &ve)
		})
	}
}

func TestUseItemBehavior_Validate_併用効果があれば空振りでも使える(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 治療に加えて回復も持つアイテムは、治せる不調が無くても回復で使える
	item := world.ECS.NewEntity()
	world.Components.Consumable.Add(item, &gc.Consumable{})
	world.Components.Remedy.Add(item, &gc.Remedy{Treats: []gc.ConditionType{gc.ConditionType("fracture")}, Potency: 150})
	world.Components.ProvidesHealing.Add(item, &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 10})

	comp := &gc.Activity{BehaviorName: gc.BehaviorUseItem, Params: &gc.UseItemParams{Target: item}}
	u := &UseItemBehavior{}
	assert.NoError(t, u.Validate(comp, player, world), "治療専用でなければ弾かない")
}
