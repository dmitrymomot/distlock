package distlock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dmitrymomot/distlock/inmem"
)

func Test_genValue(t *testing.T) {
	count := 10000
	res := map[string]struct{}{}
	for i := 0; i < count; i++ {
		v := genValue()
		if _, ok := res[string(v)]; ok {
			t.Fatalf("value already exists: %v", v)
		}
		res[string(v)] = struct{}{}
	}
	if len(res) != count {
		t.Fatalf("result values: go=%d, want=%d", len(res), count)
	}
}

func Test_manageStorageAsync(t *testing.T) {
	sd1 := inmem.New()
	sd2 := inmem.New()
	sd3 := inmem.New()
	pool := []mutexStorageDriver{sd1, sd2, sd3}
	key := "test_key"
	value := genValue()
	n := storeAsync(context.TODO(), pool, key, value, time.Second)
	if n != len(pool) {
		t.Fatalf("store result number: got=%d, want=%d", n, len(pool))
	}

	for _, sd := range pool {
		res, err := sd.Get(context.TODO(), key)
		if err != nil {
			t.Fatalf("Get(%s) = %s, got error = %v", key, value, res)
		}
	}

	sd4 := inmem.New()
	poold := []mutexStorageDriver{sd1, sd2, sd4}
	nd := deleteAsync(context.TODO(), poold, key, value)
	if nd != len(poold)-1 {
		t.Fatalf("store result number: got=%d, want=%d", nd, len(poold)-1)
	}

	// t.Fatalf("n=%d, nd=%d", n, nd)
}

func Test_defaultDelayFunc(t *testing.T) {
	d := defaultDelayFunc(1)
	w := time.Duration(100 * time.Millisecond)
	if d != w {
		t.Fatalf("defaultDelayFunc(1) = %v, want = %v", d, w)
	}

	d2 := defaultDelayFunc(0)
	w2 := time.Duration(100 * time.Millisecond)
	if d2 != w2 {
		t.Fatalf("defaultDelayFunc(0) = %v, want = %v", d2, w2)
	}

	d3 := defaultDelayFunc(10)
	w3 := time.Duration(100 * time.Millisecond)
	if d3 < w3 {
		t.Fatalf("defaultDelayFunc(0) = %v < %v", d3, w3)
	}
}

func Test_MustPanic_NoStorage(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	New()
}

func Test_MustPanic_Quorum(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	New(
		WithStorageDrivers(
			inmem.New(),
			inmem.New(),
		),
	)
}

func Test_Mutex(t *testing.T) {
	m := New(
		WithStorageDrivers(
			inmem.New(),
			inmem.New(),
			inmem.New(),
		),
		WithTries(3),
	)

	key := "test_key"
	ttl := time.Second

	if err := m.Lock(key, ttl); err != nil {
		t.Fatalf("Lock(%s, %v) = %v, want = nil", key, ttl, err)
	}

	if err := m.Lock(key, ttl); err == nil || !errors.Is(err, ErrAlreadyTaken) {
		t.Fatalf("Lock(%s, %v) = %v, want error", key, ttl, err)
	}

	// Wait for the lock expiration
	time.Sleep(ttl)

	if err := m.Lock(key, ttl); err != nil {
		t.Fatalf("Lock(%s, %v) = %v, want = nil", key, ttl, err)
	}

	if err := m.Lock(key, ttl); err == nil || !errors.Is(err, ErrAlreadyTaken) {
		t.Fatalf("Lock(%s, %v) = %v, want = %v", key, ttl, err, ErrAlreadyTaken)
	}

	if err := m.Lock(key, ttl); err == nil || !errors.Is(err, ErrAlreadyTaken) {
		t.Fatalf("Lock(%s, %v) = %v, want = %v", key, ttl, err, ErrAlreadyTaken)
	}

	if err := m.Unlock(key); err != nil {
		t.Fatalf("Unlock(%s) = %v, want = nil", key, err)
	}

	if err := m.Unlock(key); err == nil || !errors.Is(err, ErrQuorumNotDeleted) {
		t.Fatalf("Unlock(%s) = %v, want = %v", key, err, ErrQuorumNotDeleted)
	}
}
