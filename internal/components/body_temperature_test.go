package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBodyPart(t *testing.T) {
	t.Parallel()

	t.Run("String returns correct names", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			part     BodyPart
			expected string
		}{
			{BodyPartTorso, "胴体"},
			{BodyPartHead, "頭"},
			{BodyPartArms, "腕"},
			{BodyPartHands, "手"},
			{BodyPartLegs, "脚"},
			{BodyPartFeet, "足"},
		}

		for _, tt := range tests {
			t.Run(tt.expected, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.expected, tt.part.String())
			})
		}
	})

	t.Run("不正な値でpanicする", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			_ = BodyPart(99).String()
		})
	})

	t.Run("BodyPartCount is 6", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, BodyPart(6), BodyPartCount)
	})
}

func TestIsExtremity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		part     BodyPart
		expected bool
	}{
		{BodyPartTorso, false},
		{BodyPartHead, false},
		{BodyPartArms, false},
		{BodyPartHands, true},
		{BodyPartLegs, false},
		{BodyPartFeet, true},
	}

	for _, tt := range tests {
		t.Run(tt.part.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsExtremity(tt.part))
		})
	}
}
