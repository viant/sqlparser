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
	Type            string
	RawType         reflect.Type `json:"-"`
	Length          *int64
	IsUnique        bool
	IsNullable      bool
	IsAutoincrement bool
	Default         *string
	Tag             string
}
