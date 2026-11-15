package rawschema

// IsBox はタイルを 3D で高さのある箱として描くかを返す
func (t Tile) IsBox() bool {
	return t.Box != nil && *t.Box
}
