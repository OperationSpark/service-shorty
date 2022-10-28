package shorty

import (
	"errors"
)

var ErrLinkNotFound = errors.New("Link not found")
var ErrJSONUnmarshal = errors.New("Cannot parse Link JSON")
var ErrCodeInUse = errors.New("Code already in use")
