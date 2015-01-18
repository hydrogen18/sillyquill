package sillyquill_rt

import "fmt"

type UnknownColumnError struct {
	Index int
	Name  string
}

func (this UnknownColumnError) Error() string {
	return fmt.Sprintf("unknown column %q at position %d", this.Name, this.Index)
}

type RowNotUniquelyIdentifiableError struct {
	Instance interface{}
}

func (this RowNotUniquelyIdentifiableError) Error() string {
	return fmt.Sprintf("Instance of type %T not uniquely identifiable:%v",
		this.Instance,
		this.Instance)
}

type RowDoesNotExistError struct {
	Instance interface{}
}

func (this RowDoesNotExistError) Error() string {
	return fmt.Sprintf("Instance of type %T does not exist:%v", this.Instance,
		this.Instance)
}
