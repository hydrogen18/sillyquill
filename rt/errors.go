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
	Instance  interface{}
	TableName string
}

func (this RowNotUniquelyIdentifiableError) Error() string {
	return fmt.Sprintf("Instance of %T row in table %q not uniquely identifiable:%v",
		this.Instance,
		this.TableName,
		this.Instance)
}
