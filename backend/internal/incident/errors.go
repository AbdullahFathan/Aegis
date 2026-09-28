package incident

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound   = errors.New("incident not found")
	ErrValidation = errors.New("validation failed")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("conflict")
	ErrIllegal    = errors.New("illegal transition")
)

func WrapValidation(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}

func WrapIllegal(msg string) error {
	return fmt.Errorf("%w: %s", ErrIllegal, msg)
}
