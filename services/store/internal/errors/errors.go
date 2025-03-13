package errors

const (
	AlreadyClaimed             string = "already_claimed"
	AlreadyOwned               string = "already_owned"
	AlreadyRegistered          string = "already_registered"
	AssertionCreationFailed    string = "assertion-creation-failed"
	BadRequest                 string = "bad-request"
	DatabaseError              string = "database-error"
	InternalServerError        string = "internal-server-error"
	FailedToRegister           string = "failed-to-register"
	Invalid                    string = "invalid"
	InvalidChoice              string = "invalid_choice"
	InvalidField               string = "invalid-field"
	MacaroonPermissionRequired string = "macaroon-permission-required"
	MediaFileSizeTooBig        string = "media-file-size-too-big"
	MediaInvalidAspectRatio    string = "media-invalid-aspect-ratio"
	MediaInvalidResolution     string = "media-invalid-resolution"
	MediaModified              string = "media-modified"
	MediaTooManyItems          string = "media-too-many-items"
	MediaUnsupportedType       string = "media-unsupported-type"
	MissingField               string = "missing-field"
	NameNotAvailableForDispute string = "name-not-available-for-dispute"
	NameNotRegistered          string = "name-not-registered"
	RegisterWindow             string = "register_window"
	Required                   string = "required"
	ReservedName               string = "reserved_name"
	ResourceForbidden          string = "resource-forbidden"
	ResourceNotFound           string = "resource-not-found"
	ResourceNotReady           string = "resource-not-ready"
	RevokedName                string = "revoked_name"
	UserNotReady               string = "user-not-ready"
)

type Error struct {
	Code    string
	Message string
}

// Error implements the error interface for the Error struct.
func (e *Error) Error() string {
	return e.Message
}

func Is(err error, message string) bool {
	if err == nil {
		return false
	}

	if err.Error() == message {
		return true
	}

	return false
}
