package sillyquill_rt

import "fmt"

type UnknownColumnError struct {
	Index int
	Name  string
}

func (this UnknownColumnError) Error() string {
	return fmt.Sprintf("unknown column %q at position %d", this.Name, this.Index)
}
