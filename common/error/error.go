package error

import (
	"encoding/json"
	"fmt"
	"net/http"

	accountpb "github.com/idlab-discover/kebeng/services/account/proto"
	assertionpb "github.com/idlab-discover/kebeng/services/assertion/proto"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
)

const (
	AlreadyClaimed             = "already_claimed"
	AlreadyOwned               = "already_owned"
	AlreadyRegistered          = "already_registered"
	AssertionCreationFailed    = "assertion-creation-failed"
	BadRequest                 = "bad-request"
	DatabaseError              = "database-error"
	FailedToRegister           = "failed-to-register"
	FeatureDisabled            = "feature-disabled"
	InternalServerError        = "internal-server-error"
	Invalid                    = "invalid"
	InvalidChoice              = "invalid_choice"
	InvalidField               = "invalid-field"
	MacaroonPermissionRequired = "macaroon-permission-required"
	MediaFileSizeTooBig        = "media-file-size-too-big"
	MediaInvalidAspectRatio    = "media-invalid-aspect-ratio"
	MediaInvalidResolution     = "media-invalid-resolution"
	MediaModified              = "media-modified"
	MediaTooManyItems          = "media-too-many-items"
	MediaUnsupportedType       = "media-unsupported-type"
	MissingField               = "missing-field"
	NameNotAvailableForDispute = "name-not-available-for-dispute"
	NameNotRegistered          = "name-not-registered"
	NotImplemented             = "not-implemented"
	RegisterWindow             = "register_window"
	Required                   = "required"
	ReservedName               = "reserved_name"
	ResourceForbidden          = "resource-forbidden"
	ResourceNotFound           = "resource-not-found"
	ResourceNotReady           = "resource-not-ready"
	RevokedName                = "revoked_name"
	Unauthorized               = "unauthorized"
	UserNotReady               = "user-not-ready"
)

type CustomError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// helper struct
type ErrorList []CustomError

func (el *ErrorList) AddCustomError(err CustomError) {
	*el = append(*el, err)
}

func (el *ErrorList) Add(code, message string) {
	*el = append(*el, CustomError{
		Code:    code,
		Message: message,
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

func (el *ErrorList) ExtendAccountError(other []*accountpb.Error) {
	for _, err := range other {
		el.Add(err.Code, err.Message)
	}
}

func (el *ErrorList) ExtendAssertionError(other []*assertionpb.Error) {
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
	// Take the first error code in the list
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
	case DatabaseError:
		return http.StatusInternalServerError
	case FailedToRegister:
		return http.StatusInternalServerError
	case FeatureDisabled:
		return http.StatusForbidden
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
