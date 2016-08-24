package sillyquill_rt

import "github.com/hydrogen18/sillyquill/dec"
import "fmt"
import "database/sql/driver"

type Numeric struct {
	dec.Dec
}

func (this *Numeric) Scan(src interface{}) error {
	if v, ok := src.(string); ok {
		if _, ok = this.SetString(v); ok {
			return nil
		}
	}

	if v, ok := src.([]uint8); ok {
		if _, ok = this.SetString(string(v)); ok {
			return nil
		}
	}
	return fmt.Errorf("Value %v(%T) not convertible to numeric", src, src)
}

func (this Numeric) Value() (driver.Value, error) {

	return this.String(), nil
}

type NullNumeric Numeric

func (this NullNumeric) Value() (driver.Value, error) {
	return Numeric(this).Value()
}

func (this *NullNumeric) Scan(src interface{}) error {
	var v Numeric
	err := (v).Scan(src)
	if err != nil {
		return err
	}
	this.Dec = v.Dec
	return nil
}
