package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
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

	idx := pf.planData.Level.XYTileIndex(consts.Tile(x), consts.Tile(y))
	tile := pf.planData.Tiles[idx]

	// 歩行可能
	return !tile.BlockPass
}

// FindPath はBFSを使ってスタート地点からゴールまでのパスを探索する
// 上下左右の4方向移動のみサポート
func (pf *PathFinder) FindPath(startX, startY, goalX, goalY int) []consts.Coord[int] {
	width := int(pf.planData.Level.TileWidth)
	height := int(pf.planData.Level.TileHeight)

	// スタートまたはゴールが歩行不可能な場合は空のパスを返す
	if !pf.IsWalkable(startX, startY) || !pf.IsWalkable(goalX, goalY) {
		return []consts.Coord[int]{}
	}

	// 訪問済みマップ
	visited := make([][]bool, width)
	for i := range visited {
		visited[i] = make([]bool, height)
	}

	// 親ポイントマップ（パス復元用）
	parent := make([][]consts.Coord[int], width)
	for i := range parent {
		parent[i] = make([]consts.Coord[int], height)
		for j := range parent[i] {
			parent[i][j] = consts.Coord[int]{X: -1, Y: -1} // 無効値で初期化
		}
	}

	// BFS用のキュー
	queue := []consts.Coord[int]{{X: startX, Y: startY}}
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
				parent[nextX][nextY] = consts.Coord[int]{X: current.X, Y: current.Y}
				queue = append(queue, consts.Coord[int]{X: nextX, Y: nextY})
			}
		}
	}

	// パスが見つからなかった場合は空のスライスを返す
	return []consts.Coord[int]{}
}

// reconstructPath は親ポイントマップからパスを復元する
func (pf *PathFinder) reconstructPath(parent [][]consts.Coord[int], startX, startY, goalX, goalY int) []consts.Coord[int] {
	var path []consts.Coord[int]
	current := consts.Coord[int]{X: goalX, Y: goalY}

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

// countReachableFrom は指定位置からBFSで到達可能な歩行可能タイル数を返す
func (pf *PathFinder) countReachableFrom(startX, startY int) int {
	width := int(pf.planData.Level.TileWidth)
	height := int(pf.planData.Level.TileHeight)

	if !pf.IsWalkable(startX, startY) {
		return 0
	}

	visited := make([][]bool, width)
	for i := range visited {
		visited[i] = make([]bool, height)
	}

	type pos struct{ x, y int }
	queue := []pos{{startX, startY}}
	visited[startX][startY] = true
	count := 0

	directions := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		count++

		for _, d := range directions {
			nx, ny := cur.x+d[0], cur.y+d[1]
			if nx >= 0 && nx < width && ny >= 0 && ny < height && !visited[nx][ny] && pf.IsWalkable(nx, ny) {
				visited[nx][ny] = true
				queue = append(queue, pos{nx, ny})
			}
		}
	}
	return count
}

// minReachableTiles はスポーン位置から到達可能なタイルの最小数。
// 小さな孤立部屋にスポーンしてポータルが至近距離に配置される問題を防ぐ
const minReachableTiles = 100

// FindPlayerStartPosition はプレイヤーの開始位置を探す
func (pf *PathFinder) FindPlayerStartPosition() (consts.Coord[int], error) {
	planData := pf.planData

	// SpawnPointsが設定されていればそれを使用（テンプレートマップ用）
	if len(planData.SpawnPoints) > 0 {
		return consts.Coord[int]{X: planData.SpawnPoints[0].X, Y: planData.SpawnPoints[0].Y}, nil
	}

	// プロシージャルマップ用: 歩行可能でポータルに到達可能な位置を探す
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 候補位置を試す（中央から外側へ）
	attempts := []consts.Coord[int]{
		{X: width / 2, Y: height / 2},
		{X: width / 4, Y: height / 4},
		{X: 3 * width / 4, Y: height / 4},
		{X: width / 4, Y: 3 * height / 4},
		{X: 3 * width / 4, Y: 3 * height / 4},
	}

	for _, pos := range attempts {
		if pf.isValidSpawnPosition(pos.X, pos.Y) {
			return pos, nil
		}
	}

	// 見つからない場合は全体をスキャン
	for _i, tile := range planData.Tiles {
		if !tile.BlockPass {
			i := resources.TileIdx(_i)
			x, y := planData.Level.XYTileCoord(i)
			if pf.isValidSpawnPosition(int(x), int(y)) {
				return consts.Coord[int]{X: int(x), Y: int(y)}, nil
			}
		}
	}

	return consts.Coord[int]{}, fmt.Errorf("ポータルに到達可能な歩行可能タイルが見つかりません")
}

// isValidSpawnPosition は指定位置がスポーン可能かつ十分な広さに到達可能かを判定する
func (pf *PathFinder) isValidSpawnPosition(x, y int) bool {
	planData := pf.planData
	idx := planData.Level.XYTileIndex(consts.Tile(x), consts.Tile(y))
	if int(idx) >= len(planData.Tiles) || planData.Tiles[idx].BlockPass {
		return false
	}

	// NPC・アイテム・ポータルなど計画済みエンティティとの重複を防ぐ
	if planData.existPlannedEntityOnTile(x, y) {
		return false
	}

	// 到達可能なタイル数が十分であることを確認する。
	// 単一BFSで済むため、複数BFSを要するポータル到達性チェックより先に行う
	if pf.countReachableFrom(x, y) < minReachableTiles {
		return false
	}

	// NextPortalsへの到達性をチェック
	for _, portal := range planData.NextPortals {
		if !pf.IsReachable(x, y, portal.X, portal.Y) {
			return false
		}
	}

	// EscapePortalsへの到達性をチェック
	for _, portal := range planData.EscapePortals {
		if !pf.IsReachable(x, y, portal.X, portal.Y) {
			return false
		}
	}

	return true
}

// ValidatePortalReachability はプレイヤー開始位置から全ポータルへの到達性を検証する
func (pf *PathFinder) ValidatePortalReachability() error {
	// プレイヤー開始位置を取得
	playerPos, err := pf.FindPlayerStartPosition()
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
