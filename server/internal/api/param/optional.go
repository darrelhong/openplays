// Package param provides reusable custom wrapper types for huma query/header parameters.
package param

import (
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// Optional wraps a value type to distinguish "not provided" from "zero value"
// in huma query/header parameters. Implements huma.ParamWrapper and huma.ParamReactor.
//
// Usage:
//
//	type MyInput struct {
//	    Lat param.Optional[float64] `query:"lat" doc:"Latitude"`
//	}
//
//	// In handler:
//	if input.Lat.Set {
//	    fmt.Println(input.Lat.Value)
//	}
type Optional[T any] struct {
	Value T
	Set   bool
}

// Schema returns the schema of the wrapped type for OpenAPI generation.
func (o Optional[T]) Schema(r huma.Registry) *huma.Schema {
	return huma.SchemaFromType(r, reflect.TypeOf(o.Value))
}

// Receiver exposes the wrapped value field so huma can parse into it.
// Must have pointer receiver.
func (o *Optional[T]) Receiver() reflect.Value {
	return reflect.ValueOf(o).Elem().Field(0)
}

// OnParamSet is called by huma after parsing the parameter.
// Must have pointer receiver.
func (o *Optional[T]) OnParamSet(isSet bool, parsed any) {
	o.Set = isSet
}
