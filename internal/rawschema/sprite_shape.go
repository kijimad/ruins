package rawschema

// HasShape は 3D で立てる形が shape かを返す。形を省いた立て板はどの shape でもない
func (s SpriteRender) HasShape(shape SpriteShape) bool {
	return s.Shape != nil && *s.Shape == shape
}
