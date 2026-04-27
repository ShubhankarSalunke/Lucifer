package datamodel

type CustomError struct {
	err            error
	httpStatusCode int
}

func NewCustomError(err error, httpStatusCode int) *CustomError {
	return &CustomError{
		err:            err,
		httpStatusCode: httpStatusCode,
	}
}

func (e *CustomError) Error() string {
	return e.err.Error()
}

func (e *CustomError) Unwrap() error {
	return e.err
}

func (e *CustomError) GetStatusCode() int {
	return e.httpStatusCode
}
