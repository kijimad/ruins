package mapplanner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
)

// PathFinder はパスファインディング機能を提供する
type PathFinder struct {
	planData *MetaPlan
}

// NewPathFinder はPathFinderを作成する
func NewPathFinder(planData *MetaPlan) *PathFinder {
	return &PathFinder{planData: planData}
}

// IsWalkable は指定座標が歩行可能かを判定する
func (pf *PathFinder) IsWalkable(x, y int) bool {
	width := int(pf.planData.Level.TileWidth)
	height := int(pf.planData.Level.TileHeight)

	// 境界チェック
	if x < 0 || x >= width || y < 0 || y >= height {
		return false
	}

	idx := pf.planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))
	tile := pf.planData.Tiles[idx]

	// 歩行可能
	return !tile.BlockPass
}

// FindPath はBFSを使ってスタート地点からゴールまでのパスを探索する
// 上下左右の4方向移動のみサポート
func (pf *PathFinder) FindPath(startX, startY, goalX, goalY int) []Coord {
	width := int(pf.planData.Level.TileWidth)
	height := int(pf.planData.Level.TileHeight)

	// スタートまたはゴールが歩行不可能な場合は空のパスを返す
	if !pf.IsWalkable(startX, startY) || !pf.IsWalkable(goalX, goalY) {
		return []Coord{}
	}

	// 訪問済みマップ
	visited := make([][]bool, width)
	for i := range visited {
		visited[i] = make([]bool, height)
	}

	// 親ポイントマップ（パス復元用）
	parent := make([][]Coord, width)
	for i := range parent {
		parent[i] = make([]Coord, height)
		for j := range parent[i] {
			parent[i][j] = Coord{X: -1, Y: -1} // 無効値で初期化
		}
	}

	// BFS用のキュー
	queue := []Coord{{X: startX, Y: startY}}
	visited[startX][startY] = true

	// 4方向の移動方向
	directions := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

	// BFS実行
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// ゴールに到達した場合
		if current.X == goalX && current.Y == goalY {
			// パスを復元
			return pf.reconstructPath(parent, startX, startY, goalX, goalY)
		}

		// 隣接する4方向をチェック
		for _, dir := range directions {
			nextX := current.X + dir[0]
			nextY := current.Y + dir[1]

			// 境界チェックと歩行可能性チェック
			if nextX >= 0 && nextX < width && nextY >= 0 && nextY < height &&
				!visited[nextX][nextY] && pf.IsWalkable(nextX, nextY) {

				visited[nextX][nextY] = true
				parent[nextX][nextY] = Coord{X: current.X, Y: current.Y}
				queue = append(queue, Coord{X: nextX, Y: nextY})
			}
		}
	}

	// パスが見つからなかった場合は空のスライスを返す
	return []Coord{}
}

// reconstructPath は親ポイントマップからパスを復元する
func (pf *PathFinder) reconstructPath(parent [][]Coord, startX, startY, goalX, goalY int) []Coord {
	var path []Coord
	current := Coord{X: goalX, Y: goalY}

	// ゴールからスタートまで逆順にたどる
	for current.X != -1 && current.Y != -1 {
		path = append(path, current)
		if current.X == startX && current.Y == startY {
			break
		}
		current = parent[current.X][current.Y]
	}

	// パスを反転（スタートからゴールの順序にする）
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}

// IsReachable はスタート地点からゴール地点まで到達可能かを判定する
func (pf *PathFinder) IsReachable(startX, startY, goalX, goalY int) bool {
	path := pf.FindPath(startX, startY, goalX, goalY)
	return len(path) > 0
}

// ValidateConnectivity は最上列と最下列の接続性を検証する
// 最上列の歩行可能なタイルのいずれかから最下列に到達できるかをチェックする
// 橋の結合前に呼ばれる
func (pf *PathFinder) ValidateConnectivity() error {
	width := int(pf.planData.Level.TileWidth)
	height := int(pf.planData.Level.TileHeight)

	// 最上列と最下列の歩行可能タイル数を数える
	topWalkableCount := 0
	bottomWalkableCount := 0
	for x := 0; x < width; x++ {
		if pf.IsWalkable(x, 0) {
			topWalkableCount++
		}
		if pf.IsWalkable(x, height-1) {
			bottomWalkableCount++
		}
	}

	if topWalkableCount == 0 {
		return fmt.Errorf("%w: 最上列に歩行可能タイルが存在しません (width=%d, height=%d)", ErrConnectivity, width, height)
	}
	if bottomWalkableCount == 0 {
		return fmt.Errorf("%w: 最下列に歩行可能タイルが存在しません (width=%d, height=%d)", ErrConnectivity, width, height)
	}

	// 最上列の歩行可能なタイルを全て試し、いずれかから最下列に到達できるかチェック
	for topX := 0; topX < width; topX++ {
		if !pf.IsWalkable(topX, 0) {
			continue
		}
		for bottomX := 0; bottomX < width; bottomX++ {
			if !pf.IsWalkable(bottomX, height-1) {
				continue
			}
			if pf.IsReachable(topX, 0, bottomX, height-1) {
				return nil
			}
		}
	}

	return fmt.Errorf("%w: 最上列から最下列への到達不可 (width=%d, height=%d, 最上列歩行可能タイル数=%d, 最下列歩行可能タイル数=%d)",
		ErrConnectivity, width, height, topWalkableCount, bottomWalkableCount)
}

// ValidatePortalReachability はプレイヤー開始位置から全ポータルへの到達性を検証する
func (pf *PathFinder) ValidatePortalReachability() error {
	// プレイヤー開始位置を取得
	playerPos, err := pf.planData.GetPlayerStartPosition()
	if err != nil {
		return fmt.Errorf("%w: プレイヤー開始位置が設定されていません", ErrConnectivity)
	}

	// NextPortalsの到達性をチェック
	for i, portal := range pf.planData.NextPortals {
		if !pf.IsReachable(playerPos.X, playerPos.Y, portal.X, portal.Y) {
			return fmt.Errorf("%w: プレイヤー開始位置(%d,%d)からNextPortal[%d](%d,%d)への到達不可",
				ErrConnectivity, playerPos.X, playerPos.Y, i, portal.X, portal.Y)
		}
	}

	// EscapePortalsの到達性をチェック
	for i, portal := range pf.planData.EscapePortals {
		if !pf.IsReachable(playerPos.X, playerPos.Y, portal.X, portal.Y) {
			return fmt.Errorf("%w: プレイヤー開始位置(%d,%d)からEscapePortal[%d](%d,%d)への到達不可",
				ErrConnectivity, playerPos.X, playerPos.Y, i, portal.X, portal.Y)
		}
	}

	return nil
}
