package schema

// Register represent a register
type Register struct {
	Name   string
	Global bool
	Spec   string
}

type RegisterSet struct {
	Register
	TTL int
}
