package shorty

import (
	"errors"
)

var ErrLinkNotFound = errors.New("link not found")
var ErrJSONUnmarshal = errors.New("cannot parse Link JSON")
var ErrCodeInUse = errors.New("code already in use")
var ErrRelativeURL = errors.New("URL is relative")
var ErrInvalidURL = errors.New("URL improperly formatted")
