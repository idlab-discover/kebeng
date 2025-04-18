package cerror

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"

	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/lib/pq"
	"slices"
)

const (
	AlreadyClaimed             = "already-claimed"
	AlreadyOwned               = "already-owned"
	AlreadyRegistered          = "already-registered"
	AssertionCreationFailed    = "assertion-creation-failed"
	BadRequest                 = "bad-request"
	DatabaseError              = "database-error"
	FailedToRegister           = "failed-to-register"
	FeatureDisabled            = "feature-disabled"
	InternalServerError        = "internal-server-error"
	Invalid                    = "invalid"
	InvalidChoice              = "invalid-choice"
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
	RegisterWindow             = "register-window"
	Required                   = "required"
	ReservedName               = "reserved-name"
	ResourceForbidden          = "resource-forbidden"
	ResourceNotFound           = "resource-not-found"
	ResourceNotReady           = "resource-not-ready"
	RevokedName                = "revoked-name"
	Unauthorized               = "unauthorized"
	UserNotReady               = "user-not-ready"
)

type CustomError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *CustomError) GetCode() string {
	return e.Code
}

func (e *CustomError) GetMessage() string {
	return e.Message
}

type ErrorList []*CustomError

func NewCustomError(code, message string) *CustomError {
	return &CustomError{
		Code:    code,
		Message: message,
	}
}

func ConvertError(err error, message ...string) *CustomError {
	if err == nil {
		return nil
	}

	if err == sql.ErrNoRows {
		return buildCustomError(ResourceNotFound, "Resource not found", message...)
	}

	if err, ok := err.(*pq.Error); ok {
		return handlePqError(err)
	}

	// Fallback for non-PostgreSQL errors.
	return &CustomError{
		Code:    InternalServerError,
		Message: err.Error(),
	}
}

func buildCustomError(code, defaultMessage string, message ...string) *CustomError {
	msg := defaultMessage
	if len(message) > 0 {
		msg = message[0]
	}

	return &CustomError{
		Code:    code,
		Message: msg,
	}
}

func (el *ErrorList) HasError() bool {
	if el == nil {
		return false
	}
	return len(*el) > 0
}

// AddCustomError adds a custom error to the error-list.
func (el *ErrorList) AddCustomError(err *CustomError) {
	*el = append(*el, err)
}

func (el *ErrorList) Add(code, message string) {
	*el = append(*el, &CustomError{
		Code:    code,
		Message: message,
	})
}

func (el *ErrorList) Extend(other ErrorList) {
	*el = append(*el, other...)
}

func NewErrorList() *ErrorList {
	return &ErrorList{}
}

func NewError(code, message string) *ErrorList {
	el := NewErrorList()
	el.Add(code, message)
	return el
}

func (el *ErrorList) getCode() string {
	// Take the first error in the list
	first := el.getFirst()
	if first == nil {
		return ""
	}
	if ce, ok := first.(*CustomError); ok {
		return ce.Code
	}
	return ""
}

func (el *ErrorList) getFirst() any {
	if len(*el) == 0 {
		return nil
	}
	return (*el)[0]
}

// ################# PROTO ERRORS #################

func (el *ErrorList) ExtendProtoError(other []*cerrorpb.Error) {
	for _, err := range other {
		el.Add(err.Code, err.Message)
	}
}

func (el *ErrorList) ConvertToProtoErrorList() []*cerrorpb.Error {
	var errors []*cerrorpb.Error
	for _, err := range *el {
		errors = append(errors, &cerrorpb.Error{
			Code:    err.Code,
			Message: err.Message,
		})
	}
	return errors
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

func (el ErrorList) Value() (driver.Value, error) {
	if len(el) == 0 {
		return "[]", nil
	}
	return json.Marshal(el)
}

func (el *ErrorList) Scan(value interface{}) error {
	if value == nil {
		if el != nil {
			*el = ErrorList{}
		}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, el)
}

func (el *ErrorList) RemoveErrorWithCode(errorCode string) {
	for i := len(*el) - 1; i >= 0; i-- {
		currentError := (*el)[i]
		if currentError.Code == errorCode {
			*el = slices.Delete(*el, i, i+1)
		}
	}
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

func handlePqError(err *pq.Error) *CustomError {
	switch err.Code {
	case "23505": // unique violation
		return &CustomError{
			Code:    AlreadyRegistered,
			Message: err.Message,
		}
	case "23502": // not null violation
		return &CustomError{
			Code:    MissingField,
			Message: err.Message,
		}
	case "23514": // check violation
		return &CustomError{
			Code:    InvalidField,
			Message: err.Message,
		}
	case "23503": // foreign key violation
		return &CustomError{
			Code:    ResourceNotFound,
			Message: err.Message,
		}
	default:
		return &CustomError{
			Code:    DatabaseError,
			Message: err.Message,
		}
	}

}
