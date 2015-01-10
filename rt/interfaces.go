package sillyquill_rt

type Scanner interface {
	Scan(...interface{}) error
}

type Rows interface {
	Scanner
	Columns() ([]string, error)
	Next() bool
	Err() error
	Close() error
}
