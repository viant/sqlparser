package column

type Spec struct {
	Name            string
	Comments        string
	Type            string
	Nullable        bool
	Default         *string
	Key             string
	IsAutoincrement bool
}
