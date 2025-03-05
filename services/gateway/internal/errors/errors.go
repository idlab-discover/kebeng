package errors

import (
    "encoding/json"
    "fmt"


)

const (
    ResourceForbidden           string = "resource-forbidden"
    ResourceNotFound            string = "resource-not-found"
    AssertionCreationFailed     string = "assertion-creation-failed"    
    BadRequest                  string = "bad-request"                  
    FeatureDisabled             string = "feature-disabled"             
    InternalServerError         string = "internal-server-error"        
    InvalidField                string = "invalid-field"                
    MacaroonPermissionRequired  string = "macaroon-permission-required" 
    MissingField                string = "missing-field"                
    UserNotReady                string = "user-not-ready"               
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

func New() *ErrorList {
	return &ErrorList{}
}

func NewError(code, message string) *ErrorList {
    el := New()
    el.Add(code, message)
    return el
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





