package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
	smatcher "github.com/viant/sqlparser/matcher"

	"github.com/viant/parsly/matcher/option"
)

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
	keyTokenCode
	notNullToken
	createTableToken
	defaultToken
	ifNotExistsToken
	dropTableToken
	deleteCode
)

var whitespaceMatcher = parsly.NewToken(whitespaceCode, "whitespace", matcher.NewWhiteSpace())
var parenthesesMatcher = parsly.NewToken(parenthesesCode, "()", matcher.NewBlock('(', ')', '\\'))
var nextMatcher = parsly.NewToken(nextCode, ",", matcher.NewByte(','))
var asKeywordMatcher = parsly.NewToken(asKeyword, "AS", matcher.NewFragment("as", &option.Case{}))
var starTokenMatcher = parsly.NewToken(starTokenCode, "*", matcher.NewByte('*'))
var notOperatorMatcher = parsly.NewToken(notOperator, "NOT", matcher.NewFragment("not", &option.Case{}))
var nullMatcher = parsly.NewToken(nullTokenCode, "NULL", matcher.NewFragment("null", &option.Case{}))
var selectionKindMatcher = parsly.NewToken(selectionKindCode, "ALL|DISTINCT|STRUCT", matcher.NewSet([]string{
	"ALL", "DISTINCT", "STRUCT",
}, &option.Case{}))
var orderDirectionMatcher = parsly.NewToken(orderDirection, "ASC|DESC", matcher.NewSet([]string{
	"ASC", "DESC",
}, &option.Case{}))
var caseBlockMatcher = parsly.NewToken(caseBlock, "CASE", matcher.NewSeqBlock("CASE", "END"))
var commentBlockMatcher = parsly.NewToken(commentBlock, "/* */", matcher.NewSeqBlock("/*", "*/"))
var inlineCommentMatcher = parsly.NewToken(commentBlock, "--", matcher.NewSeqBlock("--", "\n"))

var selectKeywordMatcher = parsly.NewToken(selectKeyword, "SELECT", matcher.NewFragment("select", &option.Case{}))
var exceptKeywordMatcher = parsly.NewToken(exceptKeyword, "EXCEPT", matcher.NewFragment("except", &option.Case{}))
var betweenKeywordMatcher = parsly.NewToken(betweenToken, "BETWEEN", matcher.NewFragment("between", &option.Case{}))

var fromKeywordMatcher = parsly.NewToken(fromKeyword, "FROM", matcher.NewFragment("from", &option.Case{}))
var joinMatcher = parsly.NewToken(joinToken, "LEFT OUTER JOIN|LEFT JOIN|JOIN", matcher.NewSpacedSet([]string{
	"left outer join",
	"left join",
	"inner join",
	"join",
}, &option.Case{}))

var keyMatcher = parsly.NewToken(keyTokenCode, "[RANGE|HASH|PRIMARY] KEY", matcher.NewSpacedSet([]string{
	"range key",
	"hash key",
	"primary key",
}, &option.Case{}))

var onKeywordMatcher = parsly.NewToken(onKeyword, "ON", matcher.NewFragment("on", &option.Case{}))

var whereKeywordMatcher = parsly.NewToken(whereKeyword, "WHERE", matcher.NewFragment("where", &option.Case{}))
var groupByMatcher = parsly.NewToken(groupByKeyword, "GROUP BY", matcher.NewSpacedFragment("group by", &option.Case{}))
var havingKeywordMatcher = parsly.NewToken(havingKeyword, "HAVING", matcher.NewFragment("having", &option.Case{}))

var orderByKeywordMatcher = parsly.NewToken(orderByKeyword, "ORDER BY", matcher.NewSpacedFragment("order by", &option.Case{}))
var windowMatcher = parsly.NewToken(windowTokenCode, "LIMIT|OFFSET", matcher.NewSet([]string{"limit", "offset"}, &option.Case{}))

var updateKeywordMatcher = parsly.NewToken(updateKeyword, "UPDATE", matcher.NewFragment("update", &option.Case{}))
var setKeywordMatcher = parsly.NewToken(setKeyword, "SET", matcher.NewFragment("set", &option.Case{}))

var insertIntoKeywordMatcher = parsly.NewToken(insertIntoKeyword, "INSERT INTO", matcher.NewSpacedSet([]string{
	"insert into"}, &option.Case{}))

var insertValesKeywordMatcher = parsly.NewToken(insertValuesKeyword, "VALUES", matcher.NewFragment("values", &option.Case{}))

var binaryOperatorMatcher = parsly.NewToken(binaryOperator, "binary OPERATOR", matcher.NewSpacedSet([]string{"+", "!=", "=", "-", ">", "<", "=>", "=<", "*", "/", "in", "not in", "is not", "is"}, &option.Case{}))
var assignOperatorMatcher = parsly.NewToken(assignOperator, "assign OPERATOR", matcher.NewSpacedSet([]string{"="}, &option.Case{}))

var logicalOperatorMatcher = parsly.NewToken(logicalOperator, "AND|OR", matcher.NewSet([]string{"and", "or"}, &option.Case{}))
var rangeOperatorMatcher = parsly.NewToken(rangeOperator, ".. AND .. ", matcher.NewSet([]string{"and"}, &option.Case{}))

var nullKeywordMatcher = parsly.NewToken(nullKeyword, "NULL", matcher.NewFragment("null", &option.Case{}))
var boolLiteralMatcher = parsly.NewToken(boolLiteral, "true|false", matcher.NewSet([]string{"true", "false"}, &option.Case{}))
var singleQuotedStringLiteralMatcher = parsly.NewToken(singleQuotedStringLiteral, `'...'`, matcher.NewByteQuote('\'', '\\'))
var doubleQuotedStringLiteralMatcher = parsly.NewToken(doubleQuotedStringLiteral, `"..."`, matcher.NewByteQuote('\'', '\\'))
var intLiteralMatcher = parsly.NewToken(intLiteral, `INT`, smatcher.NewIntMatcher())
var numericLiteralMatcher = parsly.NewToken(numericLiteral, `NUMERIC`, matcher.NewNumber())

var identifierMatcher = parsly.NewToken(identifierCode, "IDENT", smatcher.NewIdentifier())
var selectorMatcher = parsly.NewToken(selectorTokenCode, "SELECTOR", smatcher.NewSelector())
var placeholderMatcher = parsly.NewToken(placeholderTokenCode, "SELECTOR", smatcher.NewPlaceholder())
var literalMatcher = parsly.NewToken(literalCode, "LITERAL", matcher.NewNop())
var deleteMatcher = parsly.NewToken(deleteCode, "DELETE", matcher.NewFragmentsFold([]byte("delete")))
var notNullMatcher = parsly.NewToken(notNullToken, "NOT NULL", matcher.NewSpacedSet([]string{
	"not null"}, &option.Case{}))
var ifNotExistsMatcher = parsly.NewToken(ifNotExistsToken, "IF NOT EXISTS", matcher.NewSpacedSet([]string{
	"if not exists"}, &option.Case{}))

var createTableMatcher = parsly.NewToken(createTableToken, "CREATE TABLE", matcher.NewSpacedSet([]string{
	"create table"}, &option.Case{}))

var defaultMatcher = parsly.NewToken(defaultToken, "DEFAULT", matcher.NewFragment("default", &option.Case{}))

var dropTableMatcher = parsly.NewToken(dropTableToken, "DROP TABLE", matcher.NewSpacedSet([]string{
	"drop table"}, &option.Case{}))
