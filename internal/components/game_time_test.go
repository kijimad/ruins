package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestStartOfTimeOfDayTurns(t *testing.T) {
	t.Parallel()

	for _, tod := range []TimeOfDay{TimeDawn, TimeMorning, TimeDay, TimeEvening, TimeNight, TimeMidnight} {
		gt := &GameTime{TotalTurns: StartOfTimeOfDayTurns(tod)}
		assert.Equal(t, tod, gt.GetTimeOfDay(), "開始ターンはその時間帯を指すべき")
	}
	// 昼は500ターンから始まる
	assert.Equal(t, consts.Turn(500), StartOfTimeOfDayTurns(TimeDay))
}

func TestGameTime_GetTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   TimeOfDay
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
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetTimeOfDay())
		})
	}
}

func TestGameTime_GetTemperatureModifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   int
	}{
		{"夜明けは+0°C", 0, 0},
		{"朝は+5°C", 250, 5},
		{"昼は+10°C", 500, 10},
		{"夕は+5°C", 750, 5},
		{"夜は-5°C", 1000, -5},
		{"深夜は-10°C", 1250, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetTemperatureModifier())
		})
	}
}

func TestGameTime_Advance(t *testing.T) {
	t.Parallel()

	gt := &GameTime{TotalTurns: 0}
	gt.Advance()
	assert.Equal(t, 1, int(gt.TotalTurns))

	gt.Advance()
	assert.Equal(t, 2, int(gt.TotalTurns))
}

func TestGameTime_AdvanceToNextTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		start     consts.Turn
		wantTurns consts.Turn
		wantTOD   TimeOfDay
	}{
		{"夜明けの途中から朝の開始へ", 100, 250, TimeMorning},
		{"朝の開始からちょうど昼へ", 250, 500, TimeDay},
		{"夕の途中から夜へ", 800, 1000, TimeNight},
		{"深夜からは翌日の夜明けへ折り返す", 1300, 1500, TimeDawn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.start}
			gt.AdvanceToNextTimeOfDay()
			assert.Equal(t, int(tt.wantTurns), int(gt.TotalTurns))
			assert.Equal(t, tt.wantTOD, gt.GetTimeOfDay())
		})
	}
}

// TestGameTime_AdvanceToNextTimeOfDay_常に1つ進む はどのターンから呼んでも
// 時間帯がちょうど1つ進むことを保証する。
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

func TestGameTime_GetDayNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   int
	}{
		{"ターン0は1日目", 0, 1},
		{"ターン1499は1日目", 1499, 1},
		{"ターン1500は2日目", 1500, 2},
		{"ターン3000は3日目", 3000, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetDayNumber())
		})
	}
}

func TestGameTime_ElapsedTurns(t *testing.T) {
	t.Parallel()

	// 昼開始なので原点は 500。新規ゲーム直後は経過0、そこから TotalTurns の増分ぶん増える
	assert.Equal(t, consts.Turn(500), GameStartTurns(), "昼開始の原点は500")

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   int
	}{
		{"開始直後は経過0", GameStartTurns(), 0},
		{"開始から10ターン", GameStartTurns() + 10, 10},
		{"翌日昼までで1500", GameStartTurns() + 1500, 1500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.ElapsedTurns())
		})
	}
}
