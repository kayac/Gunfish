package gunfish

import (
	"fmt"
)

// Application global variables
var (
	srvStats               Stats
	errorResponseHandler   ResponseHandler
	successResponseHandler ResponseHandler
)

// InitErrorResponseHandler initialize error response handler.
func InitErrorResponseHandler(erh ResponseHandler) error {
	if erh != nil {
		errorResponseHandler = erh
		return nil
	}
	return fmt.Errorf("Invalid response handler: %v", erh)
}

// InitSuccessResponseHandler initialize success response handler.
func InitSuccessResponseHandler(sh ResponseHandler) error {
	if sh != nil {
		successResponseHandler = sh
		return nil
	}
	return fmt.Errorf("Invalid response handler: %v", sh)
}
