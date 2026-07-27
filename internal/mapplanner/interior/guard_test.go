package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildGuardMachine_鍵は錠より手前で同じIDで結ばれる は依存グラフの解ける保証を固定する。鍵・錠・
// payload が同じ MachineID を持ち、鍵の部屋は payload の部屋より入口に近い。これで錠に着く前に必ず鍵を
// 通り、施錠戦利品が必ず解ける。
func TestBuildGuardMachine_鍵は錠より手前で同じIDで結ばれる(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 18, H: 18}
	rooms := houseRooms(PlanHouseMid(footprint, 1))
	addEntrance(footprint, rooms) // 単体テストは Site を通さないので建物入口を1つ開ける
	depths := roomDepths(rooms)

	m, ok := buildGuardMachine(5, rooms, depths, "meds", 1)
	require.True(t, ok, "複数の深さの部屋があれば machine が組める")

	assert.Equal(t, 1, m.Lock.MachineID, "錠が MachineID を持つ")
	assert.Equal(t, m.Lock.MachineID, m.Key.MachineID, "鍵と錠が同じ ID")
	assert.Equal(t, m.Lock.MachineID, m.Payload.MachineID, "payload と錠が同じ ID")
	assert.Equal(t, "keycard", m.Key.Ref, "鍵はキーカード")
	assert.Equal(t, "meds", m.Payload.Ref, "payload は指定の戦利品")

	keyDepth := roomDepthOf(rooms, depths, m.Key.Pos)
	payloadDepth := roomDepthOf(rooms, depths, m.Payload.Pos)
	require.NotEqual(t, depthUnreachable, keyDepth, "鍵が部屋の内側にある")
	require.NotEqual(t, depthUnreachable, payloadDepth, "payload が部屋の内側にある")
	assert.Less(t, keyDepth, payloadDepth, "鍵の部屋は payload の部屋より入口に近い")
}

// TestBuildGuardMachine_同じseedで完全一致する は machine の決定性を固定する。再訪一致と serde の前提。
func TestBuildGuardMachine_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 18, H: 18}
	rooms := houseRooms(PlanHouseMid(footprint, 1))
	addEntrance(footprint, rooms) // 単体テストは Site を通さないので建物入口を1つ開ける
	depths := roomDepths(rooms)

	first, ok := buildGuardMachine(5, rooms, depths, "meds", 1)
	require.True(t, ok)
	for range 5 {
		again, _ := buildGuardMachine(5, rooms, depths, "meds", 1)
		require.Equal(t, first, again, "同じ引数なら machine も完全一致する")
	}
}

// TestGuardedLoot_部屋が足りなければ非施錠へフォールバックする は rollback を固定する。深さの異なる部屋が
// 無い単室では錠も鍵も作れないので、payload だけを最奥へ非施錠、MachineID なしで置く bare へ落ちる。
func TestGuardedLoot_部屋が足りなければ非施錠へフォールバックする(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 10, H: 10}
	single := []Room{{Rect: footprint, Doorways: []Doorway{{X: 5, Y: 9}}}}

	placed := guardedLoot(3, single, "meds", 1)
	require.Len(t, placed, 1, "bare は payload だけを置く")
	assert.Equal(t, "meds", placed[0].Ref)
	assert.Equal(t, 0, placed[0].MachineID, "非施錠なので machine に属さない")
}

// roomDepthOf は座標を内側に含む部屋の入口からの距離を返す。戸口など周壁上や部屋外は depthUnreachable。
func roomDepthOf(rooms []Room, depths []roomDepth, p Vec) roomDepth {
	for i, r := range rooms {
		if r.Rect.containsInterior(p) {
			return depths[i]
		}
	}
	return depthUnreachable
}
