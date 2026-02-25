package model

import (
	"fmt"
	"reflect"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// decimalCodec is a custom BSON codec for shopspring/decimal.Decimal.
// It stores decimals as strings in MongoDB to preserve precision.
type decimalCodec struct{}

func (dc *decimalCodec) EncodeValue(_ bson.EncodeContext, w bson.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != reflect.TypeOf(decimal.Decimal{}) {
		return fmt.Errorf("decimalCodec can only encode decimal.Decimal")
	}
	d := val.Interface().(decimal.Decimal)
	return w.WriteString(d.StringFixed(2))
}

func (dc *decimalCodec) DecodeValue(_ bson.DecodeContext, r bson.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != reflect.TypeOf(decimal.Decimal{}) {
		return fmt.Errorf("decimalCodec can only decode decimal.Decimal")
	}

	var str string
	switch r.Type() {
	case bson.TypeString:
		s, err := r.ReadString()
		if err != nil {
			return err
		}
		str = s
	case bson.TypeDouble:
		f, err := r.ReadDouble()
		if err != nil {
			return err
		}
		str = fmt.Sprintf("%f", f)
	case bson.TypeInt32:
		i, err := r.ReadInt32()
		if err != nil {
			return err
		}
		str = fmt.Sprintf("%d", i)
	case bson.TypeInt64:
		i, err := r.ReadInt64()
		if err != nil {
			return err
		}
		str = fmt.Sprintf("%d", i)
	default:
		return fmt.Errorf("cannot decode BSON type %s into decimal.Decimal", r.Type())
	}

	d, err := decimal.NewFromString(str)
	if err != nil {
		return fmt.Errorf("cannot parse decimal: %w", err)
	}
	val.Set(reflect.ValueOf(d))
	return nil
}

// NewDecimalRegistry creates a BSON registry with the custom decimal codec registered.
func NewDecimalRegistry() *bson.Registry {
	reg := bson.NewRegistry()
	reg.RegisterTypeEncoder(reflect.TypeOf(decimal.Decimal{}), &decimalCodec{})
	reg.RegisterTypeDecoder(reflect.TypeOf(decimal.Decimal{}), &decimalCodec{})
	return reg
}
