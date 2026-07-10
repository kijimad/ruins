package archespike

import (
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

// camera_system.go の追従ロジックを Ark のクエリモデルへ移植したもの。
// goecs の Join(...).Visit(...) が Filter/Query にどう変換されるかを実測する。

// マーカーと追従対象のコンポーネント
type Player struct{}                                 // マーカー（ゼロサイズ）
type GridElement struct{ X, Y int }                  // 位置
type Camera struct{ X, Y, TargetX, TargetY float64 } // カメラ

const tileSize = 32.0

// cameraFollowArk は camera_system.Update の ECS 部分の移植。
// 入力(ebiten)処理は ECS ではないため省略し、Join→Query の変換のみを示す。
func cameraFollowArk(world *ecs.World) {
	// --- 旧: world.Manager.Join(Player, GridElement).Visit(...) ---
	var playerGrid *GridElement
	pq := ecs.NewFilter2[Player, GridElement](world).Query()
	for pq.Next() {
		_, grid := pq.Get() // (*Player, *GridElement)
		playerGrid = grid
		pq.Close() // 1体で十分なので打ち切る
		break
	}

	// --- 旧: world.Manager.Join(Camera).Visit(...) ---
	cq := ecs.NewFilter1[Camera](world).Query()
	for cq.Next() {
		camera := cq.Get() // *Camera（アサーション不要）
		if playerGrid != nil {
			camera.TargetX = float64(playerGrid.X)*tileSize + tileSize/2
			camera.TargetY = float64(playerGrid.Y)*tileSize + tileSize/2
		}
		camera.X = camera.TargetX
		camera.Y = camera.TargetY
	}
}

func TestCameraSystemPort(t *testing.T) {
	t.Parallel()
	world := ecs.NewWorld()

	// プレイヤー（Player + GridElement + Camera）
	playerMap := ecs.NewMap3[Player, GridElement, Camera](world)
	player := playerMap.NewEntity(&Player{}, &GridElement{X: 3, Y: 4}, &Camera{})

	// カメラだけ持つ別エンティティ（追従対象が複数でも動くことの確認）
	camMap := ecs.NewMap1[Camera](world)
	camOnly := camMap.NewEntity(&Camera{})

	// システム実行
	cameraFollowArk(world)

	// プレイヤーのカメラが (3,4) を指す
	pcam := camMap.Get(player)
	assert.InDelta(t, float64(3)*tileSize+tileSize/2, pcam.TargetX, 1e-9)
	assert.InDelta(t, float64(4)*tileSize+tileSize/2, pcam.TargetY, 1e-9)
	assert.InDelta(t, pcam.TargetX, pcam.X, 1e-9, "スナップされている")

	// カメラだけのエンティティも同じ目標に更新される
	ocam := camMap.Get(camOnly)
	assert.InDelta(t, pcam.TargetX, ocam.TargetX, 1e-9)
}
