package store

import "errors"

// ErrNotFound is returned when a pasta does not exist.
var ErrNotFound = errors.New("pasta not found")

// ErrAlreadyLiked is returned when a session tries to like a pasta it already liked.
var ErrAlreadyLiked = errors.New("already liked")
