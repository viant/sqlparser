package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
	smatcher "github.com/viant/sqlparser/matcher"

	"github.com/viant/parsly/matcher/option"
)

// Token represents a token
type Token int

const (
	whitespaceCode int = iota
	parenthesesCode
	nextCode
	identifierCode
	starTokenCode
	nullTokenCode
	notOperator
	binaryOperator
	logicalOperator
	assignOperator
	intLiteral
	numericLiteral
	boolLiteral
	nullKeyword
	singleQuotedStringLiteral
	doubleQuotedStringLiteral
	caseBlock
	betweenToken
	orderDirection
	commentBlock
	selectKeyword
	updateKeyword
	setKeyword
	insertIntoKeyword
	insertValuesKeyword
	placeholderTokenCode
	selectorTokenCode
	tableSelectorTokenCode
	asKeyword
	selectionKindCode
	exceptKeyword
	selectionKind
	fromKeyword
	onKeyword
	joinToken
	whereKeyword
	groupByKeyword
	havingKeyword
	orderByKeyword
	rangeOperator
	windowTokenCode
	literalCode
	exprToken
	keyTokenCode
	notNullToken
	createTableToken
	defaultToken
	ifNotExistsToken
	ifExistsToken
	dropTableToken
	deleteCode
	withKeyword
	unionKeyword
)

var whitespaceMatcher = parsly.NewToken(whitespaceCode, "whitespace", matcher.NewWhiteSpace())
var parenthesesMatcher = parsly.NewToken(parenthesesCode, "()", matcher.NewBlock('(', ')', '\\'))
var nextMatcher = parsly.NewToken(nextCode, ",", matcher.NewByte(','))
var asKeywordMatcher = parsly.NewToken(asKeyword, "AS", matcher.NewKeyword("as", &option.Case{}))
var starTokenMatcher = parsly.NewToken(starTokenCode, "*", matcher.NewByte('*'))
var notOperatorMatcher = parsly.NewToken(notOperator, "NOT", matcher.NewKeyword("not", &option.Case{}))
var nullMatcher = parsly.NewToken(nullTokenCode, "NULL", matcher.NewKeyword("null", &option.Case{}))
var selectionKindMatcher = parsly.NewToken(selectionKindCode, "ALL|DISTINCT|STRUCT", matcher.NewSet([]string{
	"ALL", "DISTINCT", "STRUCT",
}, &option.Case{}))
var orderDirectionMatcher = parsly.NewToken(orderDirection, "ASC|DESC", matcher.NewSet([]string{
	"ASC", "DESC",
}, &option.Case{}))
var caseBlockMatcher = parsly.NewToken(caseBlock, "CASE", matcher.NewSeqBlock("CASE", "END"))
var commentBlockMatcher = parsly.NewToken(commentBlock, "/* */", matcher.NewSeqBlock("/*", "*/"))
var inlineCommentMatcher = parsly.NewToken(commentBlock, "--", matcher.NewSeqBlock("--", "\n"))

var selectKeywordMatcher = parsly.NewToken(selectKeyword, "SELECT", matcher.NewKeyword("select", &option.Case{}))
var exceptKeywordMatcher = parsly.NewToken(exceptKeyword, "EXCEPT", matcher.NewKeyword("except", &option.Case{}))
var betweenKeywordMatcher = parsly.NewToken(betweenToken, "BETWEEN", matcher.NewKeyword("between", &option.Case{}))

var fromKeywordMatcher = parsly.NewToken(fromKeyword, "FROM", matcher.NewKeyword("from", &option.Case{}))
var joinMatcher = parsly.NewToken(joinToken, "LEFT OUTER JOIN|LEFT JOIN|JOIN", matcher.NewSpacedSet([]string{
	"left outer join",
	"cross join",
	"left join",
	"inner join",
	"join",
}, &option.Case{}))

var keyMatcher = parsly.NewToken(keyTokenCode, "[RANGE|HASH|PRIMARY] KEY", matcher.NewSpacedSet([]string{
	"range key",
	"hash key",
	"primary key",
}, &option.Case{}))

var onKeywordMatcher = parsly.NewToken(onKeyword, "ON", matcher.NewKeyword("on", &option.Case{}))

var whereKeywordMatcher = parsly.NewToken(whereKeyword, "WHERE", matcher.NewKeyword("where", &option.Case{}))
var groupByMatcher = parsly.NewToken(groupByKeyword, "GROUP BY", matcher.NewSpacedFragment("group by", &option.Case{}))
var havingKeywordMatcher = parsly.NewToken(havingKeyword, "HAVING", matcher.NewKeyword("having", &option.Case{}))

var orderByKeywordMatcher = parsly.NewToken(orderByKeyword, "ORDER BY", matcher.NewSpacedFragment("order by", &option.Case{}))
var windowMatcher = parsly.NewToken(windowTokenCode, "LIMIT|OFFSET", matcher.NewSet([]string{"limit", "offset"}, &option.Case{}))

var updateKeywordMatcher = parsly.NewToken(updateKeyword, "UPDATE", matcher.NewKeyword("update", &option.Case{}))
var setKeywordMatcher = parsly.NewToken(setKeyword, "SET", matcher.NewKeyword("set", &option.Case{}))

var insertIntoKeywordMatcher = parsly.NewToken(insertIntoKeyword, "INSERT INTO", matcher.NewSpacedSet([]string{
	"insert into"}, &option.Case{}))

var insertValesKeywordMatcher = parsly.NewToken(insertValuesKeyword, "VALUES", matcher.NewKeyword("values", &option.Case{}))

var binaryOperatorMatcher = parsly.NewToken(binaryOperator, "binary OPERATOR", matcher.NewSpacedSet([]string{"+", "!=", ">=", "<=", "=", "-", ">", "<", "*", "/", "in", "not in", "is not", "is"}, &option.Case{}))
var assignOperatorMatcher = parsly.NewToken(assignOperator, "assign OPERATOR", matcher.NewSpacedSet([]string{"="}, &option.Case{}))

var logicalOperatorMatcher = parsly.NewToken(logicalOperator, "AND|OR", matcher.NewSet([]string{"and", "or"}, &option.Case{}))
var rangeOperatorMatcher = parsly.NewToken(rangeOperator, ".. AND .. ", matcher.NewSet([]string{"and"}, &option.Case{}))

var nullKeywordMatcher = parsly.NewToken(nullKeyword, "NULL", matcher.NewKeyword("null", &option.Case{}))
var boolLiteralMatcher = parsly.NewToken(boolLiteral, "true|false", matcher.NewSet([]string{"true", "false"}, &option.Case{}))
var singleQuotedStringLiteralMatcher = parsly.NewToken(singleQuotedStringLiteral, `'...'`, matcher.NewByteQuote('\'', '\\'))
var doubleQuotedStringLiteralMatcher = parsly.NewToken(doubleQuotedStringLiteral, `"..."`, matcher.NewByteQuote('\'', '\\'))
var intLiteralMatcher = parsly.NewToken(intLiteral, `INT`, smatcher.NewIntMatcher())
var numericLiteralMatcher = parsly.NewToken(numericLiteral, `NUMERIC`, matcher.NewNumber())

var identifierMatcher = parsly.NewToken(identifierCode, "IDENT", smatcher.NewIdentifier())
var selectorMatcher = parsly.NewToken(selectorTokenCode, "SELECTOR", smatcher.NewSelector(false))
var tableMatcher = parsly.NewToken(tableSelectorTokenCode, "TABLE MATCHER", smatcher.NewSelector(true))

var placeholderMatcher = parsly.NewToken(placeholderTokenCode, "SELECTOR", smatcher.NewPlaceholder())
var literalMatcher = parsly.NewToken(literalCode, "LITERAL", matcher.NewNop())

var withKeywordMatcher = parsly.NewToken(withKeyword, "WITH", matcher.NewKeyword("with", &option.Case{}))
var unionMatcher = parsly.NewToken(unionKeyword, "UNION|UNION ALL", matcher.NewSpacedSet([]string{
	"union all",
	"union",
}, &option.Case{}))
var exprMatcher = parsly.NewToken(exprToken, ",EXPR", matcher.NewNop())

var deleteMatcher = parsly.NewToken(deleteCode, "DELETE", matcher.NewKeyword("delete", &option.Case{}))
var notNullMatcher = parsly.NewToken(notNullToken, "NOT NULL", matcher.NewSpacedSet([]string{
	"not null"}, &option.Case{}))
var ifNotExistsMatcher = parsly.NewToken(ifNotExistsToken, "IF NOT EXISTS", matcher.NewSpacedSet([]string{
	"if not exists"}, &option.Case{}))

var ifExistsMatcher = parsly.NewToken(ifExistsToken, "IF EXISTS", matcher.NewSpacedSet([]string{
	"if exists"}, &option.Case{}))

var createTableMatcher = parsly.NewToken(createTableToken, "CREATE TABLE", matcher.NewSpacedSet([]string{
	"create table"}, &option.Case{}))

var defaultMatcher = parsly.NewToken(defaultToken, "DEFAULT", matcher.NewKeyword("default", &option.Case{}))

var dropTableMatcher = parsly.NewToken(dropTableToken, "DROP TABLE", matcher.NewSpacedSet([]string{
	"drop table"}, &option.Case{}))
