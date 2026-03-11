package errors

import "fmt"

// ErrorCode 错误码
type ErrorCode int

const (
	// 成功
	ErrOK ErrorCode = 0

	// 通用错误 (1000-1999)
	ErrUnknown        ErrorCode = 1000
	ErrInvalidRequest ErrorCode = 1001
	ErrTimeout        ErrorCode = 1002
	ErrInternal       ErrorCode = 1003

	// 用户相关 (2000-2999)
	ErrUserNotFound      ErrorCode = 2001
	ErrWrongPassword     ErrorCode = 2002
	ErrUserExists        ErrorCode = 2003
	ErrNotLoggedIn       ErrorCode = 2004
	ErrInvalidSession    ErrorCode = 2005

	// 资源相关 (3000-3999)
	ErrInsufficientResources ErrorCode = 3001
	ErrInvalidResourceType   ErrorCode = 3002

	// 建筑相关 (4000-4999)
	ErrBuildingExists    ErrorCode = 4001
	ErrInvalidPosition   ErrorCode = 4002
	ErrBuildingUpgrading ErrorCode = 4003

	// 战斗相关 (5000-5999)
	ErrNoTroops      ErrorCode = 5001
	ErrInvalidTarget ErrorCode = 5002

	// 联盟相关 (6000-6999)
	ErrAlreadyInAlliance ErrorCode = 6001
	ErrNotInAlliance     ErrorCode = 6002
	ErrAllianceFull      ErrorCode = 6003
	ErrNoPermission      ErrorCode = 6004

	// 数据库相关 (9000-9999)
	ErrDatabaseError ErrorCode = 9001
	ErrNotFound      ErrorCode = 9002
)

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"` // 调试信息（不返回客户端）
}

func (e *ErrorDetail) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// 预定义错误
var (
	ErrUnknownErr        = &ErrorDetail{Code: ErrUnknown, Message: "Unknown error"}
	ErrInvalidRequestErr = &ErrorDetail{Code: ErrInvalidRequest, Message: "Invalid request"}
	ErrTimeoutErr        = &ErrorDetail{Code: ErrTimeout, Message: "Request timeout"}
	ErrInternalErr       = &ErrorDetail{Code: ErrInternal, Message: "Internal server error"}

	ErrUserNotFoundErr      = &ErrorDetail{Code: ErrUserNotFound, Message: "User not found"}
	ErrWrongPasswordErr     = &ErrorDetail{Code: ErrWrongPassword, Message: "Wrong password"}
	ErrUserExistsErr        = &ErrorDetail{Code: ErrUserExists, Message: "User already exists"}
	ErrNotLoggedInErr       = &ErrorDetail{Code: ErrNotLoggedIn, Message: "Not logged in"}
	ErrInvalidSessionErr    = &ErrorDetail{Code: ErrInvalidSession, Message: "Invalid session"}

	ErrInsufficientResourcesErr = &ErrorDetail{Code: ErrInsufficientResources, Message: "Insufficient resources"}
	ErrInvalidResourceTypeErr   = &ErrorDetail{Code: ErrInvalidResourceType, Message: "Invalid resource type"}

	ErrBuildingExistsErr    = &ErrorDetail{Code: ErrBuildingExists, Message: "Building already exists at this position"}
	ErrInvalidPositionErr   = &ErrorDetail{Code: ErrInvalidPosition, Message: "Invalid position"}
	ErrBuildingUpgradingErr = &ErrorDetail{Code: ErrBuildingUpgrading, Message: "Building is already upgrading"}

	ErrNoTroopsErr      = &ErrorDetail{Code: ErrNoTroops, Message: "No troops"}
	ErrInvalidTargetErr = &ErrorDetail{Code: ErrInvalidTarget, Message: "Invalid target"}

	ErrAlreadyInAllianceErr = &ErrorDetail{Code: ErrAlreadyInAlliance, Message: "Already in alliance"}
	ErrNotInAllianceErr     = &ErrorDetail{Code: ErrNotInAlliance, Message: "Not in alliance"}
	ErrAllianceFullErr      = &ErrorDetail{Code: ErrAllianceFull, Message: "Alliance is full"}
	ErrNoPermissionErr      = &ErrorDetail{Code: ErrNoPermission, Message: "No permission"}

	ErrDatabaseErrorErr = &ErrorDetail{Code: ErrDatabaseError, Message: "Database error"}
	ErrNotFoundErr      = &ErrorDetail{Code: ErrNotFound, Message: "Not found"}
)

// NewError 创建新错误
func NewError(code ErrorCode, message string, detail ...string) *ErrorDetail {
	err := &ErrorDetail{
		Code:    code,
		Message: message,
	}
	if len(detail) > 0 {
		err.Detail = detail[0]
	}
	return err
}

// WrapError 包装错误
func WrapError(err error, code ErrorCode, message string) *ErrorDetail {
	return &ErrorDetail{
		Code:    code,
		Message: message,
		Detail:  err.Error(),
	}
}
