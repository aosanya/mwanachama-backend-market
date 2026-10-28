package market

import "errors"

var (
	ErrInvalid        = errors.New("market: invalid")
	ErrConflict       = errors.New("market: conflict")
	ErrNotFound       = errors.New("market: not found")
	ErrForbidden      = errors.New("market: forbidden")
	ErrAlreadyAsked   = errors.New("market: already asked")
	ErrPosterNotReady = errors.New("market: poster not approved")
	ErrPosterAway     = errors.New("market: poster is away")
)
