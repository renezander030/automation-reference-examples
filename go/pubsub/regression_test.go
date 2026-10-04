package pubsub

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"net"
	"testing"
	"time"
)

func TestFailedRetainedAndClosed(t *testing.T) {
	c := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: time.Millisecond, Dialer: func(context.Context, string, string) (net.Conn, error) { return nil, fmt.Errorf("offline") }})
	defer c.Close()
	p := NewBufferedPublisher(c, "x", 1, time.Hour)
	if p.Publish(context.Background(), "hello") == nil {
		t.Fatal("expected failure")
	}
	if len(p.buffer) != 1 {
		t.Fatal("batch lost")
	}
	p.Close(context.Background())
	if p.Publish(context.Background(), "new") == nil || p.PublishImmediate(context.Background(), "new") == nil {
		t.Fatal("closed success")
	}
}
func TestMarshalRejected(t *testing.T) {
	c := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer c.Close()
	p := NewBufferedPublisher(c, "x", 2, time.Hour)
	if p.Publish(context.Background(), make(chan int)) == nil || len(p.buffer) != 0 {
		t.Fatal("invalid event accepted")
	}
}
