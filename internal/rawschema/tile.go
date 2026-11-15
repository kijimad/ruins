package rawschema

// IsSolid はタイルを高さのある塊で描くかを返す
func (t Tile) IsSolid() bool {
	return t.Solid != nil && *t.Solid
}
