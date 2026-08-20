package consts

// Nerd Fonts アイコン定数
// https://www.nerdfonts.com/cheat-sheet
const (
	// 矢印
	IconArrowLeft  = "\uf060"
	IconArrowRight = "\uf061"
	IconArrowUp    = "\uf062"
	IconArrowDown  = "\uf063"

	// カーソル
	IconCursor = "\uf054" // fa-chevron-right

	// 操作
	IconEnter    = "\uf148"
	IconCheck    = "\uf00c"
	IconClose    = "\uf00d"
	IconSave     = "\uf0c7"
	IconKeyEnter = "\U000f0311" // md-keyboard_return
	IconKeyEsc   = "\U000f12b7" // md-keyboard_esc
	IconKeySpace = "\U000f1050" // md-keyboard_space
	IconKeyTab   = "\U000f0312" // md-keyboard_tab
	IconKeyShift = "\U000f0636" // md-apple_keyboard_shift
	IconKeyDot   = "\U000f09df" // md-circle_small。. キー

	// UI
	IconHome     = "\uf015"
	IconSettings = "\uf013"
	IconMenu     = "\uf0c9"
	IconSearch   = "\uf002"

	// 単位
	IconDegree   = "\u2103"     // 摂氏記号 ℃
	IconCurrency = "\U000f1aaf" // 通貨記号 nf-md-heating_coil
	IconKg       = "\u338f"     // ㎏
	IconG        = "g"          // グラム。単一の合字がないため素の g を使う
	IconMg       = "\u338e"     // ㎎

	// ゲーム
	IconHeart    = "\uf004"
	IconStar     = "\uf005"
	IconShield   = "\uf132"
	IconSword    = "\uf71c"
	IconSkull    = "\uf54c"
	IconFlask    = "\uf0c3"
	IconFire     = "\uf06d"
	IconBolt     = "\uf0e7"
	IconLeaf     = "\uf06c"
	IconDroplet  = "\uf043"
	IconMoon     = "\uf186"
	IconSun      = "\uf185"
	IconUser     = "\uf007"
	IconUsers    = "\uf0c0"
	IconBag      = "\uf290"
	IconMap      = "\uf279"
	IconCompass  = "\uf14e"
	IconFlag     = "\uf024"
	IconClock    = "\uf017"
	IconWarning  = "\uf071"
	IconInfo     = "\uf129"
	IconQuestion = "\uf128"
	IconCube     = "\uf1b2" // fa-cube
)

// キー一覧の箱に入れる文字グリフの基点。箱は描画側が自前で敷くため、中身は箱なしの
// plain 変種を使う。どちらも連続配置であることを実グリフのコードポイントで確認済み
const (
	// IconKeyAlphaBase は md-alpha_a。a からの差分を足す
	IconKeyAlphaBase rune = 0xF0AEE
	// IconKeyDigitBase は md-numeric_0。数字の値を足す
	IconKeyDigitBase rune = 0xF0B39
)
