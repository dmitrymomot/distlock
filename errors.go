package locker

import "errors"

// Predefined package errors
var (
	ErrAlreadyTaken     = errors.New("the lock key is already in use")
	ErrQuorumNotDeleted = errors.New("could not unlock quorum")
)
