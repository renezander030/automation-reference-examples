package config

import (
	"testing"
)

func TestMalformedAndNonpositive(t *testing.T) {
	for _, v := range []string{"bad", "0", "-1", "9223372036854775807"} {
		t.Setenv("POLL_INTERVAL_SEC", v)
		if _, err := Load(); err == nil {
			t.Fatal(v)
		}
	}
	t.Setenv("POLL_INTERVAL_SEC", "10")
	t.Setenv("ENABLE_TLS", "garbage")
	if _, err := Load(); err == nil {
		t.Fatal("invalid bool")
	}
}
func TestValid(t *testing.T) {
	t.Setenv("ENABLE_TLS", "true")
	c, err := Load()
	if err != nil || !c.EnableTLS || c.PollInterval <= 0 {
		t.Fatal(c, err)
	}
}
