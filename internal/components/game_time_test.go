package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

// TestStartOfTimeOfDayTurns は暦ターンから時間帯の開始位置を導けることを確認する。
// 暦では0が夜明けの開始で、昼は500ターンから始まる
func TestStartOfTimeOfDayTurns(t *testing.T) {
	t.Parallel()

	for _, tod := range []TimeOfDay{TimeDawn, TimeMorning, TimeDay, TimeEvening, TimeNight, TimeMidnight} {
		assert.Equal(t, tod, timeOfDayAt(StartOfTimeOfDayTurns(tod)), "その時間帯の開始ターンはその時間帯を指すべき")
	}
	assert.Equal(t, consts.Turn(500), StartOfTimeOfDayTurns(TimeDay))
}

// TestTimeOfDayAt は暦ターンから時間帯を導く純関数を確認する。暦では0が夜明け
func TestTimeOfDayAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		calendarTurn consts.Turn
		expected     TimeOfDay
	}{
		{"ターン0は夜明け", 0, TimeDawn},
		{"ターン249は夜明け", 249, TimeDawn},
		{"ターン250は朝", 250, TimeMorning},
		{"ターン499は朝", 499, TimeMorning},
		{"ターン500は昼", 500, TimeDay},
		{"ターン749は昼", 749, TimeDay},
		{"ターン750は夕", 750, TimeEvening},
		{"ターン999は夕", 999, TimeEvening},
		{"ターン1000は夜", 1000, TimeNight},
		{"ターン1249は夜", 1249, TimeNight},
		{"ターン1250は深夜", 1250, TimeMidnight},
		{"ターン1499は深夜", 1499, TimeMidnight},
		{"ターン1500は夜明け（2日目）", 1500, TimeDawn},
		{"ターン3000は夜明け（3日目）", 3000, TimeDawn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, timeOfDayAt(tt.calendarTurn))
		})
	}
}

// TestDayNumberAt は暦ターンから経過日数を導く純関数を確認する。1日目から始まる
func TestDayNumberAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		calendarTurn consts.Turn
		expected     int
	}{
		{"ターン0は1日目", 0, 1},
		{"ターン1499は1日目", 1499, 1},
		{"ターン1500は2日目", 1500, 2},
		{"ターン3000は3日目", 3000, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, dayNumberAt(tt.calendarTurn))
		})
	}
}

// TestTemperatureModifier は時間帯ごとの気温修正値を確認する純関数のテスト
func TestTemperatureModifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tod      TimeOfDay
		expected int
	}{
		{TimeDawn, 0},
		{TimeMorning, 5},
		{TimeDay, 10},
		{TimeEvening, 5},
		{TimeNight, -5},
		{TimeMidnight, -10},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, temperatureModifier(tt.tod), "%s の気温修正", tt.tod)
	}
}

// TestGameTime_startsAtNoon は新規ゲームの GameTime が経過0・昼・1日目から始まることを確認する。
// TotalTurns は経過ターンで0始まり。昼開始は時刻導出のオフセットで表すので、初期ターンを底上げしない
func TestGameTime_startsAtNoon(t *testing.T) {
	t.Parallel()

	gt := &GameTime{} // 新規ゲームの初期値
	assert.Equal(t, consts.Turn(0), gt.TotalTurns, "経過ターンは0始まり")
	assert.Equal(t, TimeDay, gt.GetTimeOfDay(), "昼から始まる")
	assert.Equal(t, 1, gt.GetDayNumber(), "1日目から始まる")
	assert.Equal(t, 10, gt.GetTemperatureModifier(), "昼の気温修正")
}

func TestGameTime_Advance(t *testing.T) {
	t.Parallel()

	gt := &GameTime{TotalTurns: 0}
	gt.Advance()
	assert.Equal(t, 1, int(gt.TotalTurns))

	gt.Advance()
	assert.Equal(t, 2, int(gt.TotalTurns))
}

// TestGameTime_AdvanceToNextTimeOfDay は次の時間帯の開始ターンまで進むことを確認する。
// start は経過ターン。昼開始のオフセットは turnsPerTimeOfDay の倍数なので、
// 経過ターンを次の区切りへ丸めれば必ず時間帯境界に着く
func TestGameTime_AdvanceToNextTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		start     consts.Turn
		wantTurns consts.Turn
	}{
		{"区切りの途中から次の区切りへ", 100, 250},
		{"区切りちょうどから次の区切りへ", 250, 500},
		{"区切りの途中から次の区切りへ2", 800, 1000},
		{"日をまたぐ区切りへ", 1300, 1500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.start}
			gt.AdvanceToNextTimeOfDay()
			assert.Equal(t, int(tt.wantTurns), int(gt.TotalTurns))
		})
	}
}

// TestGameTime_AdvanceToNextTimeOfDay_常に1つ進む はどの経過ターンから呼んでも
// 時間帯がちょうど1つ進むことを保証する。昼開始のオフセットは一定なので前後の差は変わらない
func TestGameTime_AdvanceToNextTimeOfDay_常に1つ進む(t *testing.T) {
	t.Parallel()

	for start := consts.Turn(0); start < turnsPerDay*2; start += 37 {
		gt := &GameTime{TotalTurns: start}
		before := gt.GetTimeOfDay()
		gt.AdvanceToNextTimeOfDay()
		after := gt.GetTimeOfDay()
		want := TimeOfDay((int(before) + 1) % 6)
		assert.Equal(t, want, after, "start=%d では1つ次の時間帯になるべき", start)
	}
}
