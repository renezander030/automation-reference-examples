package ratelimiter

import (
	"testing"
	"time"
)

func TestBurstAndValidation(t *testing.T) {
	l := New(2, time.Second, 5)
	for i := 0; i < 5; i++ {
		if !l.Allow() {
			t.Fatal("initial burst")
		}
	}
	if l.Allow() {
		t.Fatal("exhausted")
	}
	if !l.allowAt(5, l.last.Add(3*time.Second)) {
		t.Fatal("refill must restore burst=5")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("zero interval accepted")
		}
	}()
	New(1, 0, 1)
}
func TestConcurrent(t *testing.T) {
	l := New(1, time.Hour, 10)
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() { done <- l.Allow() }()
	}
	n := 0
	for i := 0; i < 100; i++ {
		if <-done {
			n++
		}
	}
	if n != 10 {
		t.Fatal(n)
	}
}
