package rawschema

// IsWall はタイルが壁かを返す
func (t Tile) IsWall() bool {
	return t.Wall != nil && *t.Wall
}
