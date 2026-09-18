package expr

import "github.com/viant/sqlparser/node"

// Call represents a call. Named arguments are *Binary nodes with Op "=>",
// a *Raw argument name in X, and the argument expression in Y.
type Call struct {
	X    node.Node
	Args []node.Node
	Raw  string
}
