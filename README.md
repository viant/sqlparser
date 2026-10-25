# sqlparser
SQL Parser

BigQuery `QUALIFY` conditions are represented by `query.Select.QualifyClause`.
The existing `query.Select.Qualify` field continues to represent `WHERE`.
Inline window expressions use `expr.Window`, including `PARTITION BY`,
`ORDER BY`, and `ROWS`/`RANGE` frames with integer or placeholder offsets.
Named window declarations and references (`WINDOW w AS (...)`, `OVER w`)
are not supported.

Traversal includes window expressions, QUALIFY conditions, parsed CTEs, and
derived tables. Window results are computed expressions rather than direct
scalar-column lineage. `Stringifier{PreserveWindow: true}` retains LIMIT/OFFSET
as before; inline OVER specifications and QUALIFY are always rendered.
