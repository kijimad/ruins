package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFillRoom_歩行可能な床が戸口から全て到達できる は、配置が部屋を分断せず戸口から全床へ届くことを
// 固定する。到達性は有界インスタンスで必須。塞がった床はその seed で永久に塞がるので配置成立条件にする。
func TestFillRoom_歩行可能な床が戸口から全て到達できる(t *testing.T) {
	t.Parallel()

	room, content := storeRoom(), storeContent()
	for s := range uint64(40) {
		placed := FillRoom(s, room, content)
		blocked := blockingTiles(placed)
		reached := reachableFloor(room, blocked)
		for _, tile := range room.Rect.interiorTiles() {
			if blocked[tile] {
				continue
			}
			require.Truef(t, reached[tile], "床 %v が戸口から到達できる (seed=%d)", tile, s)
		}
	}
}

// TestReachableFloor_家具の壁で奥が孤立すると到達しない は、検査が実際に分断を検知することを確認する。
// 検査が常に真を返すなら不変条件テストが自明に通ってしまう退行を防ぐ。
func TestReachableFloor_家具の壁で奥が孤立すると到達しない(t *testing.T) {
	t.Parallel()

	// 5x5 の部屋。戸口は下辺。内側 y=2 の行を家具で横一列に塞ぎ、奥(上側)を孤立させる
	room := Room{Rect: Rect{X: 0, Y: 0, W: 5, H: 5}, Doorways: []Doorway{{X: 2, Y: 4}}}
	blocked := map[Vec]bool{{X: 1, Y: 2}: true, {X: 2, Y: 2}: true, {X: 3, Y: 2}: true}

	reached := reachableFloor(room, blocked)

	assert.True(t, reached[Vec{X: 2, Y: 3}], "戸口側の床は到達する")
	assert.False(t, reached[Vec{X: 2, Y: 1}], "家具の壁の奥は到達しない")
}
