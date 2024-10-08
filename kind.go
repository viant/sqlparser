package sqlparser

import (
	"strings"
	"unicode"
)

const (
	KindUnknown       = "unknown"
	KindSelect        = "select"
	KindLoad          = "load"
	KindInsert        = "insert"
	KindUpdate        = "update"
	KindMerge         = "merge"
	KindDelete        = "delete"
	KindRegisterType  = "register type"
	KindRegisterSet   = "register set"
	KindCreateTable   = "create table"
	KindDropTable     = "drop table"
	KindCreateIndex   = "create index"
	KindDropIndex     = "drop index"
	KindTruncateTable = "truncate table"
)

// Kind represents the type of SQL statement.
type Kind string

// IsUnknown returns true if the Kind is KindUnknown.
func (k Kind) IsUnknown() bool {
	return k == KindUnknown
}

// IsInsert returns true if the Kind is KindInsert.
func (k Kind) IsInsert() bool {
	return k == KindInsert
}

// IsSelect returns true if the Kind is KindSelect.
func (k Kind) IsSelect() bool {
	return k == KindSelect
}

// IsUpdate returns true if the Kind is KindUpdate.
func (k Kind) IsUpdate() bool {
	return k == KindUpdate
}

// IsLoad returns true if the Kind is KindLoad.
func (k Kind) IsLoad() bool {
	return k == KindLoad
}

// IsMerge returns true if the Kind is KindMerge.
func (k Kind) IsMerge() bool {
	return k == KindMerge
}

// IsDelete returns true if the Kind is KindDelete.
func (k Kind) IsDelete() bool {
	return k == KindDelete
}

// IsRegisterType returns true if the Kind is KindRegisterType.
func (k Kind) IsRegisterType() bool {
	return k == KindRegisterType
}

// IsRegisterSet returns true if the Kind is KindRegisterSet.
func (k Kind) IsRegisterSet() bool {
	return k == KindRegisterSet
}

// IsCreateTable returns true if the Kind is KindCreateTable.
func (k Kind) IsCreateTable() bool {
	return k == KindCreateTable
}

// IsDropTable returns true if the Kind is KindDropTable.
func (k Kind) IsDropTable() bool {
	return k == KindDropTable
}

func ParseKind(SQL string) Kind {
	SQL = removeSQLComments(SQL)
	normalizedSQL := strings.TrimSpace(SQL)
	if len(normalizedSQL) < 2 {
		return KindUnknown
	}
	firstToken := strings.ToLower(normalizedSQL[0:2])
	secondToken := ""
	secondPart := ""

	if index := strings.Index(normalizedSQL, " "); index != -1 {
		for i := index; i < len(normalizedSQL); i++ {
			if unicode.IsLetter(rune(normalizedSQL[i])) {
				secondToken = strings.ToLower(normalizedSQL[i : i+1])
				if i+1 < len(normalizedSQL) {
					secondPart = normalizedSQL[i+1:]
				}
				break
			}
		}
	}

	thirdToken := ""
	if index := strings.Index(secondPart, " "); index != -1 {
		for i := index; i < len(secondPart); i++ {
			if unicode.IsLetter(rune(secondPart[i])) {
				thirdToken = strings.ToLower(secondPart[i : i+1])
				break
			}
		}
	}

	strings.ToLower(strings.TrimSpace(SQL)[0:1])
	switch firstToken[0] {
	case 's', 'w': //select, with
		return KindSelect
	case 'i': //insert
		return KindInsert
	case 'u': //update
		return KindUpdate
	case 'l': //load
		return KindLoad
	case 'm': //merge
		return KindMerge
	case 'd': //delete or drop
		switch firstToken[1] {
		case 'e': //delete
			return KindDelete
		case 'r': //drop
			if len(secondToken) == 0 {
				return KindUnknown
			}
			switch secondToken[0] {
			case 't': //drop table
				return KindDropTable

			case 'i':
				return KindDropIndex
			}
		}
	case 't':
		if len(secondToken) == 0 {
			return KindUnknown
		}
		switch secondToken[0] {
		case 't': //truncate
			return KindTruncateTable
		}

	case 'r': //register
		if len(secondToken) == 0 {
			return KindUnknown
		}
		switch secondToken[0] {
		case 't': //register type
			return KindRegisterType
		case 's': //register set
			return KindRegisterSet
		case 'g': //g global
			if len(thirdToken) == 0 {
				return KindUnknown
			}
			switch thirdToken[0] {
			case 't': //register type (global type)
				return KindRegisterType
			case 's': //register set (global set)
				return KindRegisterSet
			}
		}
	case 'c': //create
		if len(secondToken) == 0 {
			return KindUnknown
		}
		switch secondToken[0] {
		case 't': //create table
			return KindCreateTable
		case 'i':
			return KindCreateIndex
		}
		switch thirdToken[0] {
		case 'i':
			return KindCreateIndex
		}
	}
	return KindUnknown
}
