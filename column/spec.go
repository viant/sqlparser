package column

//Spec represents column specifiction
type Spec struct {
	Name            string
	Comments        string
	Type            string
	Nullable        bool
	Default         *string
	Key             string
	Tag             string
	IsAutoincrement bool
}
