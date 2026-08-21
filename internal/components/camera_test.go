package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCameraNormalizeOrbit(t *testing.T) {
	t.Parallel()

	t.Run("3D値を持たないセーブのゼロ値は既定の視点へ戻る", func(t *testing.T) {
		t.Parallel()
		// Dist だけ直すと Pitch がゼロのまま残り、真横から見る絵になる。3値まとめて戻す
		c := Camera{}
		c.NormalizeOrbit()
		assert.Equal(t, 0, c.Orient)
		assert.InDelta(t, CameraDefaultPitch, c.Pitch, 1e-9)
		assert.InDelta(t, CameraDefaultDist, c.Dist, 1e-9)
	})

	t.Run("可動域内の値はそのまま保たれる", func(t *testing.T) {
		t.Parallel()
		c := Camera{Orient: 3, Pitch: 0.9, Dist: 10}
		c.NormalizeOrbit()
		assert.Equal(t, 3, c.Orient)
		assert.InDelta(t, 0.9, c.Pitch, 1e-9)
		assert.InDelta(t, 10.0, c.Dist, 1e-9)
	})

	t.Run("Distが可動域を超えると既定の視点へ戻る", func(t *testing.T) {
		t.Parallel()
		c := Camera{Orient: 3, Pitch: 0.9, Dist: CameraMaxDist + 1}
		c.NormalizeOrbit()
		assert.Equal(t, 0, c.Orient)
		assert.InDelta(t, CameraDefaultDist, c.Dist, 1e-9)
	})

	t.Run("Pitchは可動域へ丸める", func(t *testing.T) {
		t.Parallel()
		low := Camera{Pitch: -1, Dist: CameraDefaultDist}
		low.NormalizeOrbit()
		assert.InDelta(t, CameraMinPitch, low.Pitch, 1e-9)

		high := Camera{Pitch: 99, Dist: CameraDefaultDist}
		high.NormalizeOrbit()
		assert.InDelta(t, CameraMaxPitch, high.Pitch, 1e-9)
	})

	t.Run("Orientは負でも環に収まる", func(t *testing.T) {
		t.Parallel()
		c := Camera{Orient: -1, Pitch: CameraDefaultPitch, Dist: CameraDefaultDist}
		c.NormalizeOrbit()
		assert.Equal(t, CameraOrientCount-1, c.Orient)
	})
}
