package models


const Nil = ServerError("server_error: nil")

type ServerError string

func (e ServerError) Error() string { return string(e) }


const (
	USER_NOT_AUTHENTICATED = ServerError("user not authenticated")
	SERVER_ERROR = ServerError("not able to complete")
	INVALID_CREDENTIALS= ServerError("invalid username or password")
	INACTIVE_USER = ServerError("User access is disabled")
	FORM_ERROR=ServerError("Invalid input")
	UNAUTHORIZED = ServerError("Not allowed")
	USER_ALREADY_EXISTS = ServerError("User with given username/email already exists")
	DUPLICATE_ENTRY = ServerError("Duplicate entry")
	INVALID_ENTRY = ServerError("Invalid entry")
)
