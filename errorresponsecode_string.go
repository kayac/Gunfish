// generated by stringer -type ErrorResponseCode error.go; DO NOT EDIT

package gunfish

import "fmt"

const _ErrorResponseCode_name = "PayloadEmptyPayloadTooLargeBadTopicTopicDisallowedBadMessageIDBadExpirationDateBadPriorityMissingDeviceTokenBadDeviceTokenDeviceTokenNotForTopicUnregisteredDuplicateHeadersBadCertificateEnvironmentBadCertificateForbiddenBadPathMethodNotAllowedTooManyRequestsIdleTimeoutShutdownInternalServerErrorServiceUnavailableMissingTopic"

var _ErrorResponseCode_index = [...]uint16{0, 12, 27, 35, 50, 62, 79, 90, 108, 122, 144, 156, 172, 197, 211, 220, 227, 243, 258, 269, 277, 296, 314, 326}

func (i ErrorResponseCode) String() string {
	if i < 0 || i >= ErrorResponseCode(len(_ErrorResponseCode_index)-1) {
		return fmt.Sprintf("ErrorResponseCode(%d)", i)
	}
	return _ErrorResponseCode_name[_ErrorResponseCode_index[i]:_ErrorResponseCode_index[i+1]]
}
