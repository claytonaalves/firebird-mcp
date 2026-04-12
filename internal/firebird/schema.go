package firebird

import "fmt"

// QuerySpec holds a SQL query and its positional arguments.
type QuerySpec struct {
	SQL  string
	Args []any
}

func TableQuery(pattern, detailLevel string, limit int) QuerySpec {
	switch detailLevel {
	case "summary":
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT
  TRIM(r.RDB$RELATION_NAME) AS name,
  (SELECT COUNT(*) FROM RDB$RELATION_FIELDS f
   WHERE f.RDB$RELATION_NAME = r.RDB$RELATION_NAME
     AND (f.RDB$SYSTEM_FLAG = 0 OR f.RDB$SYSTEM_FLAG IS NULL)) AS column_count,
  CAST(r.RDB$DESCRIPTION AS VARCHAR(255)) AS comment
FROM RDB$RELATIONS r
WHERE (r.RDB$SYSTEM_FLAG = 0 OR r.RDB$SYSTEM_FLAG IS NULL)
  AND r.RDB$VIEW_BLR IS NULL
  AND TRIM(r.RDB$RELATION_NAME) LIKE ?
ORDER BY r.RDB$RELATION_NAME
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	case "full":
		// Returns one row per column; caller groups by table_name.
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT
  TRIM(r.RDB$RELATION_NAME)                    AS table_name,
  TRIM(f.RDB$FIELD_NAME)                       AS col_name,
  TRIM(t.RDB$TYPE_NAME)                        AS col_type,
  f.RDB$NULL_FLAG                              AS not_null,
  CAST(f.RDB$DEFAULT_SOURCE AS VARCHAR(255))   AS col_default,
  CAST(f.RDB$DESCRIPTION AS VARCHAR(255))      AS col_description,
  CAST(r.RDB$DESCRIPTION AS VARCHAR(255))      AS tbl_comment
FROM RDB$RELATIONS r
JOIN RDB$RELATION_FIELDS f ON f.RDB$RELATION_NAME = r.RDB$RELATION_NAME
JOIN RDB$FIELDS fld ON fld.RDB$FIELD_NAME = f.RDB$FIELD_SOURCE
LEFT JOIN RDB$TYPES t ON t.RDB$FIELD_NAME = 'RDB$FIELD_TYPE'
                      AND t.RDB$TYPE = fld.RDB$FIELD_TYPE
WHERE (r.RDB$SYSTEM_FLAG = 0 OR r.RDB$SYSTEM_FLAG IS NULL)
  AND r.RDB$VIEW_BLR IS NULL
  AND (f.RDB$SYSTEM_FLAG = 0 OR f.RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(r.RDB$RELATION_NAME) LIKE ?
ORDER BY r.RDB$RELATION_NAME, f.RDB$FIELD_POSITION
ROWS 1 TO %d`, limit*200),
			Args: []any{pattern},
		}
	default: // "names"
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT TRIM(RDB$RELATION_NAME) AS name
FROM RDB$RELATIONS
WHERE (RDB$SYSTEM_FLAG = 0 OR RDB$SYSTEM_FLAG IS NULL)
  AND RDB$VIEW_BLR IS NULL
  AND TRIM(RDB$RELATION_NAME) LIKE ?
ORDER BY RDB$RELATION_NAME
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	}
}

func TableIndexQuery(pattern string) QuerySpec {
	return QuerySpec{
		SQL: `SELECT
  TRIM(i.RDB$RELATION_NAME)          AS table_name,
  TRIM(i.RDB$INDEX_NAME)             AS idx_name,
  i.RDB$UNIQUE_FLAG                  AS is_unique,
  TRIM(c.RDB$CONSTRAINT_TYPE)        AS constraint_type,
  TRIM(s.RDB$FIELD_NAME)             AS col_name,
  s.RDB$FIELD_POSITION               AS col_pos
FROM RDB$INDICES i
JOIN RDB$INDEX_SEGMENTS s ON s.RDB$INDEX_NAME = i.RDB$INDEX_NAME
LEFT JOIN RDB$RELATION_CONSTRAINTS c ON c.RDB$INDEX_NAME = i.RDB$INDEX_NAME
WHERE (i.RDB$SYSTEM_FLAG = 0 OR i.RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(i.RDB$RELATION_NAME) LIKE ?
ORDER BY i.RDB$RELATION_NAME, i.RDB$INDEX_NAME, s.RDB$FIELD_POSITION`,
		Args: []any{pattern},
	}
}

func ViewQuery(pattern, detailLevel string, limit int) QuerySpec {
	switch detailLevel {
	case "summary":
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT
  TRIM(r.RDB$RELATION_NAME) AS name,
  (SELECT COUNT(*) FROM RDB$RELATION_FIELDS f
   WHERE f.RDB$RELATION_NAME = r.RDB$RELATION_NAME
     AND (f.RDB$SYSTEM_FLAG = 0 OR f.RDB$SYSTEM_FLAG IS NULL)) AS column_count,
  CAST(r.RDB$DESCRIPTION AS VARCHAR(255)) AS comment
FROM RDB$RELATIONS r
WHERE (r.RDB$SYSTEM_FLAG = 0 OR r.RDB$SYSTEM_FLAG IS NULL)
  AND r.RDB$VIEW_BLR IS NOT NULL
  AND TRIM(r.RDB$RELATION_NAME) LIKE ?
ORDER BY r.RDB$RELATION_NAME
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	case "full":
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT
  TRIM(r.RDB$RELATION_NAME)                    AS table_name,
  TRIM(f.RDB$FIELD_NAME)                       AS col_name,
  TRIM(t.RDB$TYPE_NAME)                        AS col_type,
  f.RDB$NULL_FLAG                              AS not_null,
  CAST(f.RDB$DEFAULT_SOURCE AS VARCHAR(255))   AS col_default,
  CAST(f.RDB$DESCRIPTION AS VARCHAR(255))      AS col_description,
  CAST(r.RDB$DESCRIPTION AS VARCHAR(255))      AS tbl_comment,
  CAST(r.RDB$VIEW_SOURCE AS VARCHAR(2000))     AS view_source
FROM RDB$RELATIONS r
JOIN RDB$RELATION_FIELDS f ON f.RDB$RELATION_NAME = r.RDB$RELATION_NAME
JOIN RDB$FIELDS fld ON fld.RDB$FIELD_NAME = f.RDB$FIELD_SOURCE
LEFT JOIN RDB$TYPES t ON t.RDB$FIELD_NAME = 'RDB$FIELD_TYPE'
                      AND t.RDB$TYPE = fld.RDB$FIELD_TYPE
WHERE (r.RDB$SYSTEM_FLAG = 0 OR r.RDB$SYSTEM_FLAG IS NULL)
  AND r.RDB$VIEW_BLR IS NOT NULL
  AND (f.RDB$SYSTEM_FLAG = 0 OR f.RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(r.RDB$RELATION_NAME) LIKE ?
ORDER BY r.RDB$RELATION_NAME, f.RDB$FIELD_POSITION
ROWS 1 TO %d`, limit*200),
			Args: []any{pattern},
		}
	default: // "names"
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT TRIM(RDB$RELATION_NAME) AS name
FROM RDB$RELATIONS
WHERE (RDB$SYSTEM_FLAG = 0 OR RDB$SYSTEM_FLAG IS NULL)
  AND RDB$VIEW_BLR IS NOT NULL
  AND TRIM(RDB$RELATION_NAME) LIKE ?
ORDER BY RDB$RELATION_NAME
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	}
}

func ColumnQuery(pattern, detailLevel string, limit int) QuerySpec {
	switch detailLevel {
	case "summary", "full":
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT
  TRIM(f.RDB$FIELD_NAME)                       AS name,
  TRIM(f.RDB$RELATION_NAME)                    AS table_name,
  TRIM(t.RDB$TYPE_NAME)                        AS type,
  f.RDB$NULL_FLAG                              AS not_null,
  CAST(f.RDB$DEFAULT_SOURCE AS VARCHAR(255))   AS col_default,
  CAST(f.RDB$DESCRIPTION AS VARCHAR(255))      AS description
FROM RDB$RELATION_FIELDS f
JOIN RDB$RELATIONS r ON r.RDB$RELATION_NAME = f.RDB$RELATION_NAME
JOIN RDB$FIELDS fld ON fld.RDB$FIELD_NAME = f.RDB$FIELD_SOURCE
LEFT JOIN RDB$TYPES t ON t.RDB$FIELD_NAME = 'RDB$FIELD_TYPE'
                      AND t.RDB$TYPE = fld.RDB$FIELD_TYPE
WHERE (f.RDB$SYSTEM_FLAG = 0 OR f.RDB$SYSTEM_FLAG IS NULL)
  AND (r.RDB$SYSTEM_FLAG = 0 OR r.RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(f.RDB$FIELD_NAME) LIKE ?
ORDER BY f.RDB$RELATION_NAME, f.RDB$FIELD_POSITION
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	default: // "names"
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT TRIM(f.RDB$FIELD_NAME) AS name
FROM RDB$RELATION_FIELDS f
JOIN RDB$RELATIONS r ON r.RDB$RELATION_NAME = f.RDB$RELATION_NAME
WHERE (f.RDB$SYSTEM_FLAG = 0 OR f.RDB$SYSTEM_FLAG IS NULL)
  AND (r.RDB$SYSTEM_FLAG = 0 OR r.RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(f.RDB$FIELD_NAME) LIKE ?
ORDER BY f.RDB$RELATION_NAME, f.RDB$FIELD_POSITION
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	}
}

func ProcedureQuery(pattern, detailLevel string, limit int) QuerySpec {
	switch detailLevel {
	case "summary":
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT
  TRIM(p.RDB$PROCEDURE_NAME)                   AS name,
  p.RDB$PROCEDURE_INPUTS                       AS input_params,
  p.RDB$PROCEDURE_OUTPUTS                      AS output_params,
  CAST(p.RDB$DESCRIPTION AS VARCHAR(255))      AS description
FROM RDB$PROCEDURES p
WHERE (p.RDB$SYSTEM_FLAG = 0 OR p.RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(p.RDB$PROCEDURE_NAME) LIKE ?
ORDER BY p.RDB$PROCEDURE_NAME
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	case "full":
		// Returns one row per parameter; caller groups by proc_name.
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT
  TRIM(p.RDB$PROCEDURE_NAME)                   AS proc_name,
  TRIM(pp.RDB$PARAMETER_NAME)                  AS param_name,
  pp.RDB$PARAMETER_TYPE                        AS param_direction,
  TRIM(t.RDB$TYPE_NAME)                        AS param_type,
  CAST(p.RDB$DESCRIPTION AS VARCHAR(255))      AS description
FROM RDB$PROCEDURES p
LEFT JOIN RDB$PROCEDURE_PARAMETERS pp ON pp.RDB$PROCEDURE_NAME = p.RDB$PROCEDURE_NAME
LEFT JOIN RDB$FIELDS fld ON fld.RDB$FIELD_NAME = pp.RDB$FIELD_SOURCE
LEFT JOIN RDB$TYPES t ON t.RDB$FIELD_NAME = 'RDB$FIELD_TYPE'
                      AND t.RDB$TYPE = fld.RDB$FIELD_TYPE
WHERE (p.RDB$SYSTEM_FLAG = 0 OR p.RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(p.RDB$PROCEDURE_NAME) LIKE ?
ORDER BY p.RDB$PROCEDURE_NAME, pp.RDB$PARAMETER_TYPE, pp.RDB$PARAMETER_NUMBER
ROWS 1 TO %d`, limit*50),
			Args: []any{pattern},
		}
	default: // "names"
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT TRIM(RDB$PROCEDURE_NAME) AS name
FROM RDB$PROCEDURES
WHERE (RDB$SYSTEM_FLAG = 0 OR RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(RDB$PROCEDURE_NAME) LIKE ?
ORDER BY RDB$PROCEDURE_NAME
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	}
}

func FunctionQuery(pattern, detailLevel string, limit int) QuerySpec {
	switch detailLevel {
	case "summary", "full":
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT
  TRIM(f.RDB$FUNCTION_NAME)                    AS name,
  CAST(f.RDB$DESCRIPTION AS VARCHAR(255))      AS description
FROM RDB$FUNCTIONS f
WHERE (f.RDB$SYSTEM_FLAG = 0 OR f.RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(f.RDB$FUNCTION_NAME) LIKE ?
ORDER BY f.RDB$FUNCTION_NAME
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	default: // "names"
		return QuerySpec{
			SQL: fmt.Sprintf(`SELECT TRIM(RDB$FUNCTION_NAME) AS name
FROM RDB$FUNCTIONS
WHERE (RDB$SYSTEM_FLAG = 0 OR RDB$SYSTEM_FLAG IS NULL)
  AND TRIM(RDB$FUNCTION_NAME) LIKE ?
ORDER BY RDB$FUNCTION_NAME
ROWS 1 TO %d`, limit),
			Args: []any{pattern},
		}
	}
}
