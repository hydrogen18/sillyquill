package sillyquill_rt

type Scanner interface {
	Scan(...interface{}) error
}
