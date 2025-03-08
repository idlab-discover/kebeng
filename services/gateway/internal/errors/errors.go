package errors

import (
	"encoding/json"
	"fmt"
	"net/http"

	storepb "github.com/idlab-discover/kebeng/services/store/proto"
)

const (
	AlreadyClaimed             string = "already_claimed"
	AlreadyOwned               string = "already_owned"
	AlreadyRegistered          string = "already_registered"
	AssertionCreationFailed    string = "assertion-creation-failed"
	BadRequest                 string = "bad-request"
    FailedToRegister           string = "failed-to-register"
	InternalServerError        string = "internal-server-error"
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
    NotImplemented             string = "not-implemented"
	RegisterWindow             string = "register_window"
	Required                   string = "required"
	ReservedName               string = "reserved_name"
	ResourceForbidden          string = "resource-forbidden"
	ResourceNotFound           string = "resource-not-found"
	ResourceNotReady           string = "resource-not-ready"
	RevokedName                string = "revoked_name"
    Unauthorized               string = "unauthorized"
	UserNotReady               string = "user-not-ready"
)

// helper struct
type ErrorList []map[string]string

func (el *ErrorList) Add(code, message string) {
	*el = append(*el, map[string]string{
		"code":    code,
		"message": message,
	})
}

func (el *ErrorList) Extend(other ErrorList) {
	*el = append(*el, other...)
}

func (el *ErrorList) ExtendStoreError(other []*storepb.Error) {
	for _, err := range other {
		el.Add(err.Code, err.Message)
	}
}

func New() *ErrorList {
	return &ErrorList{}
}

func NewError(code, message string) *ErrorList {
	el := New()
	el.Add(code, message)
	return el
}

func (el *ErrorList) getCode() string {
	first := el.getFirst()
	if first == nil {
		return ""
	}
	return first.(map[string]string)["code"]
}

func (el *ErrorList) getFirst() any {
	if len(*el) == 0 {
		return nil
	}
	return (*el)[0]
}

func FormatBindError(err error) string {
	// check for type error
	if ute, ok := err.(*json.UnmarshalTypeError); ok {
		return fmt.Sprintf("Expected %s to be of type %s. Got: %s", ute.Field, ute.Type.String(), ute.Value)
	}

	// check for syntax error
	if se, ok := err.(*json.SyntaxError); ok {
		return fmt.Sprintf("Syntax error at byte offset %d", se.Offset)
	}

	// default
	return err.Error()
}

func (el *ErrorList) GetHTTPStatus() int {
	switch el.getCode() {
	case AlreadyClaimed:
		return http.StatusConflict
	case AlreadyOwned:
		return http.StatusConflict
	case AlreadyRegistered:
		return http.StatusConflict
	case AssertionCreationFailed:
		return http.StatusInternalServerError
	case BadRequest:
		return http.StatusBadRequest
    case FailedToRegister:
        return http.StatusInternalServerError
	case InternalServerError:
		return http.StatusInternalServerError
	case Invalid:
		return http.StatusBadRequest
	case InvalidChoice:
		return http.StatusBadRequest
	case InvalidField:
		return http.StatusBadRequest
	case MacaroonPermissionRequired:
		return http.StatusForbidden
	case MediaFileSizeTooBig:
		return http.StatusRequestEntityTooLarge
	case MediaInvalidAspectRatio:
		return http.StatusBadRequest
	case MediaInvalidResolution:
		return http.StatusBadRequest
	case MediaModified:
		return http.StatusConflict
	case MediaTooManyItems:
		return http.StatusBadRequest
	case MediaUnsupportedType:
		return http.StatusUnsupportedMediaType
	case MissingField:
		return http.StatusBadRequest
	case NameNotAvailableForDispute:
		return http.StatusConflict
	case NameNotRegistered:
		return http.StatusNotFound
    case NotImplemented:
        return http.StatusNotImplemented
	case RegisterWindow:
		return http.StatusBadRequest
	case Required:
		return http.StatusBadRequest
	case ReservedName:
		return http.StatusConflict
	case ResourceForbidden:
		return http.StatusForbidden
	case ResourceNotFound:
		return http.StatusNotFound
	case ResourceNotReady:
		return http.StatusServiceUnavailable
	case RevokedName:
		return http.StatusGone
    case Unauthorized:
        return http.StatusUnauthorized
	case UserNotReady:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
