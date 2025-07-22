package bizflow

type Meta struct {
	RetryNum       int
	IsRetryErr     map[error]bool
	IsNoneRetryErr map[error]bool
	AbortErr       map[error]bool
}