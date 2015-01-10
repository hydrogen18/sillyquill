package sillyquill_rt

type UnknownColumnError struct {
	Index int
	Name  string
}

func (this UnknownColumnError) Error() string {
	return "unknown column"
}
