package components

// 部位定義
// 体温・健康状態システムで使用する部位の定義

// BodyPart は体の部位を表す
type BodyPart int

// 体の部位定数
const (
	BodyPartHead  BodyPart = iota // 頭
	BodyPartTorso                 // 胴体
	BodyPartArms                  // 腕
	BodyPartHands                 // 手
	BodyPartLegs                  // 脚
	BodyPartFeet                  // 足
	BodyPartCount                 // 部位数 = 6
)

// String は部位名を返す
func (bp BodyPart) String() string {
	switch bp {
	case BodyPartHead:
		return "頭"
	case BodyPartTorso:
		return "胴体"
	case BodyPartArms:
		return "腕"
	case BodyPartHands:
		return "手"
	case BodyPartLegs:
		return "脚"
	case BodyPartFeet:
		return "足"
	default:
		panic("不正なBodyPart値")
	}
}

// IsExtremity は末端部位かどうかを返す
// 凍傷は末端部位のみで発症する
func IsExtremity(part BodyPart) bool {
	return part == BodyPartHands || part == BodyPartFeet
}
