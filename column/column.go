package column

import (
	"reflect"
)

//Spec represents column spec
type Spec struct {
	Namespace       string
	Name            string
	Key             string
	Alias           string
	Except          []string
	Expression      string
	Comments        string
	DataType        string
	Type            reflect.Type `json:"-"`
	Length          int
	Nullable        bool
	Default         *string
	Tag             string
	IsAutoincrement bool
}
