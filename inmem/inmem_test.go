package inmem

import (
	"context"
	"testing"
	"time"
)

func Test_InMemStorage(t *testing.T) {
	k := "test_key"
	v := []byte("test_value")
	ttl := time.Second
	s := New()

	if v, err := s.Get(context.TODO(), k); err == nil {
		t.Fatalf("inmem.Get(%s) = %v; want error", k, v)
	}

	if err := s.Set(context.TODO(), k, v, ttl); err != nil {
		t.Fatalf("inmem.Set(%s, %v) = %v; want = nil", k, v, err)
	}

	if v, err := s.Get(context.TODO(), k); err != nil {
		t.Fatalf("inmem.Get(%s) = %v, %v; want = %v, nil", k, v, err, v)
	}

	if err := s.Set(context.TODO(), k, v, ttl); err == nil {
		t.Fatalf("inmem.Set(%s, %v) = nil; want error", k, v)
	}

	if err := s.Delete(context.TODO(), k); err != nil {
		t.Fatalf("inmem.Delete(%s) = %v; want = nil", k, err)
	}

	if err := s.Delete(context.TODO(), k); err == nil {
		t.Fatalf("inmem.Delete(%s) = nil; want error", k)
	}
}
