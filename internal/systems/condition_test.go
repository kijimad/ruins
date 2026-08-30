package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spawnWithCondition は指定した不調を1つ持つ実体を作る
func spawnWithCondition(world w.World, part gc.BodyPart, cond gc.HealthCondition) (ecs.Entity, *gc.HealthStatus) {
	entity := world.ECS.NewEntity()
	hs := &gc.HealthStatus{}
	cond.Severity = gc.TimerToSeverity(cond.Timer)
	hs.Parts[part].SetCondition(cond)
	world.Components.HealthStatus.Add(entity, hs)
	return entity, hs
}

func TestConditionSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("未治療の骨折は保持する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, hs := spawnWithCondition(world, gc.BodyPartArms, gc.HealthCondition{Type: gc.ConditionFracture, Timer: 60})

		require.NoError(t, (&ConditionSystem{}).Update(world))

		cond := hs.Parts[gc.BodyPartArms].GetCondition(gc.ConditionFracture)
		require.NotNil(t, cond)
		assert.InDelta(t, 60, cond.Timer, 1e-9, "未治療の骨折は悪化も回復もしない")
	})

	t.Run("治療済みの骨折は質ぶん回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// RecoverPer=3、TendQuality=200%なので 3*2=6 減る
		_, hs := spawnWithCondition(world, gc.BodyPartArms, gc.HealthCondition{Type: gc.ConditionFracture, Timer: 60, TendQuality: 200})

		require.NoError(t, (&ConditionSystem{}).Update(world))

		cond := hs.Parts[gc.BodyPartArms].GetCondition(gc.ConditionFracture)
		require.NotNil(t, cond)
		assert.InDelta(t, 54, cond.Timer, 1e-9)
	})

	t.Run("発症前の掠り傷は自然に癒える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, hs := spawnWithCondition(world, gc.BodyPartArms, gc.HealthCondition{Type: gc.ConditionFracture, Timer: 10})

		require.NoError(t, (&ConditionSystem{}).Update(world))

		cond := hs.Parts[gc.BodyPartArms].GetCondition(gc.ConditionFracture)
		require.NotNil(t, cond)
		assert.InDelta(t, 9, cond.Timer, 1e-9, "発症前は未治療でも僅かに減る")
	})

	t.Run("未治療の病気は悪化する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// WorsenPer=2 増える
		_, hs := spawnWithCondition(world, gc.BodyPartTorso, gc.HealthCondition{Type: gc.ConditionLiverIllness, Timer: 40})

		require.NoError(t, (&ConditionSystem{}).Update(world))

		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionLiverIllness)
		require.NotNil(t, cond)
		assert.InDelta(t, 42, cond.Timer, 1e-9)
	})

	t.Run("治療済みの病気は回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// 能力も満腹度も無いので代謝100%。RecoverPer=3、TendQuality=100%なので 3 減る
		_, hs := spawnWithCondition(world, gc.BodyPartTorso, gc.HealthCondition{Type: gc.ConditionLiverIllness, Timer: 60, TendQuality: 100})

		require.NoError(t, (&ConditionSystem{}).Update(world))

		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionLiverIllness)
		require.NotNil(t, cond)
		assert.InDelta(t, 57, cond.Timer, 1e-9)
	})

	t.Run("重症の病気は毎ターンHPを削る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity, _ := spawnWithCondition(world, gc.BodyPartTorso, gc.HealthCondition{Type: gc.ConditionLiverIllness, Timer: 80})
		world.Components.HP.Add(entity, &gc.HP{Current: 30, Max: 30})

		require.NoError(t, (&ConditionSystem{}).Update(world))

		// HPDamage=2 削られる
		assert.Equal(t, 28, world.Components.HP.Get(entity).Current)
	})

	t.Run("Timerが0になると不調は消える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// Timer=1 に RecoverPer=3 の回復が来るので0でクランプされ除去される
		_, hs := spawnWithCondition(world, gc.BodyPartArms, gc.HealthCondition{Type: gc.ConditionFracture, Timer: 1, TendQuality: 100})

		require.NoError(t, (&ConditionSystem{}).Update(world))

		assert.Nil(t, hs.Parts[gc.BodyPartArms].GetCondition(gc.ConditionFracture), "回復しきった不調は消える")
	})
}

func TestConditionCatalog_扱う不調を網羅しHypothermiaを含まない(t *testing.T) {
	t.Parallel()

	// ConditionSystem が扱う怪我と病気はカタログに載せる。登録漏れは動作不全になる
	for _, ct := range []gc.ConditionType{gc.ConditionFracture, gc.ConditionLaceration, gc.ConditionLiverIllness} {
		_, ok := conditionCatalog[ct]
		assert.True(t, ok, "%s はカタログに登録されているべき", gc.ConditionTypeDisplayName(ct))
	}

	// 低体温は TemperatureSystem が担うのでカタログに載せない
	_, ok := conditionCatalog[gc.ConditionHypothermia]
	assert.False(t, ok, "低体温はカタログに載せない")
}
