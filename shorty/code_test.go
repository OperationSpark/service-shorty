package shorty

import (
	"testing"
)

func TestCreateCode(t *testing.T) {
	t.Run("creates a random 10-character code", func(t *testing.T) {
		code := CreateCode()

		if len(code) != 10 {
			t.Fatalf("expected %s to be 10 characters", code)
		}
	})
}
