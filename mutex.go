package locker

import (
	"bytes"
	"context"
	"math/rand"
	"sync"
	"time"
)

type (
	// Mutex / mutual exclusion lock
	Mutex struct {
		pool         []mutexStorageDriver
		tries        uint
		quorum       int
		genValueFunc func() []byte
		delayFunc    func(tries int) time.Duration
		kv           sync.Map
	}

	// a lock value and expiration time
	lockVal struct {
		value []byte
		until time.Time
	}

	// Option of mutex
	Option func(*Mutex)

	// the interface of a driver to manage lock keys
	mutexStorageDriver interface {
		Set(ctx context.Context, key string, value []byte, ttl time.Duration) error // Should return error if a key is already exists
		Get(ctx context.Context, key string) ([]byte, error)
		Delete(ctx context.Context, key string) error
	}
)

// New mutex instance with options
func New(opt ...Option) *Mutex {
	m := &Mutex{
		tries:        1,
		genValueFunc: genValue,
		delayFunc:    defaultDelayFunc,
	}
	for _, o := range opt {
		o(m)
	}
	if m.pool == nil {
		panic("add at least one storage driver in pool")
	}
	if len(m.pool)%2 != 1 {
		panic("must be odd number of storage nodes to prevent split brain")
	}
	m.quorum = (len(m.pool) / 2) + 1
	return m
}

// Lock func
// Returns error if the lock is already in use.
func (m *Mutex) Lock(key string, ttl time.Duration) error {
	return m.LockContext(nil, key, ttl)
}

// LockContext func is the same as Lock but has context :)
// Returns error if the lock is already in use.
func (m *Mutex) LockContext(ctx context.Context, key string, ttl time.Duration) error {
	val := m.genValueFunc()
	for i := 0; i < int(m.tries); i++ {
		if i != 0 {
			time.Sleep(m.delayFunc(i))
		}

		start := time.Now()

		n := storeAsync(ctx, m.pool, key, val, ttl)

		now := time.Now()
		until := now.Add(ttl - now.Sub(start))

		if n >= m.quorum && now.Before(until) {
			m.kv.Store(key, lockVal{
				value: val,
				until: until,
			})
			return nil
		}

		deleteAsync(ctx, m.pool, key, val)
	}

	return ErrAlreadyTaken
}

// Unlock func
// Returns error if the lock is already in use.
func (m *Mutex) Unlock(key string) error {
	return m.UnlockContext(nil, key)
}

// UnlockContext func is the same as Unlock but has context :)
// Returns error if the lock is already in use.
func (m *Mutex) UnlockContext(ctx context.Context, key string) error {
	if value, ok := m.kv.Load(key); ok {
		if v, ok := value.(lockVal); ok {
			n := deleteAsync(ctx, m.pool, key, v.value)
			if n < m.quorum {
				return ErrQuorumNotDeleted
			}
		}
	}
	return nil
}

// Stores a lock key into each storage
func storeAsync(ctx context.Context, pool []mutexStorageDriver, key string, value []byte, ttl time.Duration) int {
	// try to store lock key into each storage in pool
	ch := make(chan bool)
	for _, sd := range pool {
		go func(s mutexStorageDriver) {
			err := s.Set(ctx, key, value, ttl)
			ch <- (err == nil)
		}(sd)
	}

	// count all successful storing attempt
	n := 0
	for range pool {
		r := <-ch
		if r {
			n++
		}
	}

	return n
}

// Delete a lock key from each storage
func deleteAsync(ctx context.Context, pool []mutexStorageDriver, key string, value []byte) int {
	// try to delete lock key from each storage in pool
	ch := make(chan bool)
	for _, sd := range pool {
		go func(s mutexStorageDriver) {
			v, err := s.Get(ctx, key)
			if err != nil {
				ch <- false
				return
			}
			// delete only records with the same values to avoid release another lock
			if bytes.Compare(v, value) == 0 {
				err := s.Delete(ctx, key)
				ch <- (err == nil)
			} else {
				ch <- false
			}
		}(sd)
	}

	// count all successful deleting attempt
	n := 0
	for range pool {
		r := <-ch
		if r {
			n++
		}
	}

	return n
}

// generates random value
func genValue() []byte {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return []byte("")
	}
	return b
}

// generates pseudo-random delay time
// to avoid run all pods at the same time
// after failed first lock attempting
func defaultDelayFunc(tries int) time.Duration {
	if tries < 1 {
		tries = 1
	}
	rn := rand.Intn(tries)
	if rn == 0 {
		rn = 1
	}
	return time.Duration(rn * 100 * int(time.Millisecond))
}
