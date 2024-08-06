package common

import "testing"

func TestRandomString(t *testing.T) {
	s := RandomString(10)
	if len(s) != 10 {
		t.Errorf("RandomString(10) failed")
		t.Failed()
	}
	t.Logf("RandomString(10): %s", s)
}
