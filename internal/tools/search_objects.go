package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	idb "github.com/claytonaalves/firebird-mcp/internal/db"
	"github.com/claytonaalves/firebird-mcp/internal/firebird"
)

type SearchParams struct {
	ObjectType  string `json:"object_type"`
	Pattern     string `json:"pattern"`
	DetailLevel string `json:"detail_level"`
	Limit       int    `json:"limit"`
}

type SearchResult struct {
	Count   int              `json:"count"`
	Results []map[string]any `json:"results"`
}

func SearchObjects(ctx context.Context, db *sql.DB, rawArgs json.RawMessage) (any, error) {
	p := SearchParams{
		Pattern:     "%",
		DetailLevel: "names",
		Limit:       100,
	}
	if err := json.Unmarshal(rawArgs, &p); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if p.Pattern == "" {
		p.Pattern = "%"
	}
	if p.DetailLevel == "" {
		p.DetailLevel = "names"
	}
	if p.Limit <= 0 {
		p.Limit = 100
	}
	if p.Limit > 1000 {
		p.Limit = 1000
	}

	switch p.ObjectType {
	case "table":
		return searchTables(ctx, db, p)
	case "view":
		return searchViews(ctx, db, p)
	case "column":
		return searchColumns(ctx, db, p)
	case "procedure":
		return searchProcedures(ctx, db, p)
	case "function":
		return searchFunctions(ctx, db, p)
	default:
		return nil, fmt.Errorf("unknown object_type: %s", p.ObjectType)
	}
}

func searchTables(ctx context.Context, db *sql.DB, p SearchParams) (any, error) {
	spec := firebird.TableQuery(p.Pattern, p.DetailLevel, p.Limit)
	cols, rows, err := idb.RunQuery(ctx, db, spec.SQL, spec.Args...)
	if err != nil {
		return nil, err
	}

	if p.DetailLevel == "names" {
		results := rowsToMaps(cols, rows)
		return SearchResult{Count: len(results), Results: results}, nil
	}

	if p.DetailLevel == "summary" {
		results := rowsToMaps(cols, rows)
		enrichRowCounts(ctx, db, results)
		return SearchResult{Count: len(results), Results: results}, nil
	}

	// "full"
	tableMap, tableOrder := groupTableColumns(rows)

	idxSpec := firebird.TableIndexQuery(p.Pattern)
	_, idxRows, err := idb.RunQuery(ctx, db, idxSpec.SQL, idxSpec.Args...)
	if err != nil {
		return nil, err
	}
	mergeIndexes(tableMap, idxRows)

	counts := fetchRowCounts(ctx, db, tableOrder)
	for name, count := range counts {
		tableMap[name]["row_count"] = count
	}

	results := make([]map[string]any, 0, len(tableOrder))
	for _, name := range tableOrder {
		results = append(results, tableMap[name])
	}
	return SearchResult{Count: len(results), Results: results}, nil
}

func searchViews(ctx context.Context, db *sql.DB, p SearchParams) (any, error) {
	spec := firebird.ViewQuery(p.Pattern, p.DetailLevel, p.Limit)
	cols, rows, err := idb.RunQuery(ctx, db, spec.SQL, spec.Args...)
	if err != nil {
		return nil, err
	}

	if p.DetailLevel == "names" {
		results := rowsToMaps(cols, rows)
		return SearchResult{Count: len(results), Results: results}, nil
	}

	if p.DetailLevel == "summary" {
		results := rowsToMaps(cols, rows)
		enrichRowCounts(ctx, db, results)
		return SearchResult{Count: len(results), Results: results}, nil
	}

	// "full"
	tableMap, tableOrder := groupTableColumns(rows)

	counts := fetchRowCounts(ctx, db, tableOrder)
	for name, count := range counts {
		tableMap[name]["row_count"] = count
	}

	results := make([]map[string]any, 0, len(tableOrder))
	for _, name := range tableOrder {
		results = append(results, tableMap[name])
	}
	return SearchResult{Count: len(results), Results: results}, nil
}

func searchColumns(ctx context.Context, db *sql.DB, p SearchParams) (any, error) {
	spec := firebird.ColumnQuery(p.Pattern, p.DetailLevel, p.Limit)
	cols, rows, err := idb.RunQuery(ctx, db, spec.SQL, spec.Args...)
	if err != nil {
		return nil, err
	}
	results := rowsToMaps(cols, rows)
	return SearchResult{Count: len(results), Results: results}, nil
}

func searchProcedures(ctx context.Context, db *sql.DB, p SearchParams) (any, error) {
	spec := firebird.ProcedureQuery(p.Pattern, p.DetailLevel, p.Limit)
	cols, rows, err := idb.RunQuery(ctx, db, spec.SQL, spec.Args...)
	if err != nil {
		return nil, err
	}

	if p.DetailLevel != "full" {
		results := rowsToMaps(cols, rows)
		return SearchResult{Count: len(results), Results: results}, nil
	}

	// group by procedure name: proc_name, param_name, param_direction, param_type, description
	procMap := map[string]map[string]any{}
	procOrder := []string{}
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		name := valStr(row[0])
		if _, exists := procMap[name]; !exists {
			procOrder = append(procOrder, name)
			proc := map[string]any{
				"name":       name,
				"parameters": []map[string]any{},
			}
			if len(row) > 4 {
				if desc := valStr(row[4]); desc != "" {
					proc["description"] = desc
				}
			}
			procMap[name] = proc
		}
		if len(row) > 1 && row[1] != nil {
			dir := "input"
			if len(row) > 2 {
				if n, ok := toInt(row[2]); ok && n == 1 {
					dir = "output"
				}
			}
			param := map[string]any{
				"name":      valStr(row[1]),
				"direction": dir,
			}
			if len(row) > 3 && row[3] != nil {
				if t := valStr(row[3]); t != "" {
					param["type"] = strings.ToLower(t)
				}
			}
			params := procMap[name]["parameters"].([]map[string]any)
			procMap[name]["parameters"] = append(params, param)
		}
	}

	results := make([]map[string]any, 0, len(procOrder))
	for _, name := range procOrder {
		results = append(results, procMap[name])
	}
	return SearchResult{Count: len(results), Results: results}, nil
}

func searchFunctions(ctx context.Context, db *sql.DB, p SearchParams) (any, error) {
	spec := firebird.FunctionQuery(p.Pattern, p.DetailLevel, p.Limit)
	cols, rows, err := idb.RunQuery(ctx, db, spec.SQL, spec.Args...)
	if err != nil {
		return nil, err
	}
	results := rowsToMaps(cols, rows)
	return SearchResult{Count: len(results), Results: results}, nil
}

// fetchRowCounts runs SELECT COUNT(*) for each table name and returns name→count.
// Table names come from RDB$RELATIONS (not user input), so direct interpolation is safe.
func fetchRowCounts(ctx context.Context, db *sql.DB, names []string) map[string]int64 {
	counts := make(map[string]int64, len(names))
	for _, name := range names {
		_, rows, err := idb.RunQuery(ctx, db, "SELECT COUNT(*) FROM "+name)
		if err != nil || len(rows) == 0 {
			continue
		}
		if n, ok := rows[0][0].(int64); ok {
			counts[name] = n
		}
	}
	return counts
}

// enrichRowCounts adds row_count to each result map (which must have a "name" key).
func enrichRowCounts(ctx context.Context, db *sql.DB, results []map[string]any) {
	names := make([]string, 0, len(results))
	for _, r := range results {
		if name, ok := r["name"].(string); ok {
			names = append(names, name)
		}
	}
	counts := fetchRowCounts(ctx, db, names)
	for _, r := range results {
		if name, ok := r["name"].(string); ok {
			if n, ok := counts[name]; ok {
				r["row_count"] = n
			}
		}
	}
}

// rowsToMaps converts flat query results to maps, keyed by lowercase column name.
// Nil values and empty strings are omitted.
func rowsToMaps(cols []string, rows [][]any) []map[string]any {
	lower := make([]string, len(cols))
	for i, c := range cols {
		lower[i] = strings.ToLower(c)
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m := map[string]any{}
		for i, v := range row {
			if v == nil {
				continue
			}
			if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
				continue
			}
			if b, ok := v.([]byte); ok {
				if strings.TrimSpace(string(b)) == "" {
					continue
				}
				m[lower[i]] = strings.TrimSpace(string(b))
				continue
			}
			m[lower[i]] = v
		}
		out = append(out, m)
	}
	return out
}

// groupTableColumns groups flat column rows (from TableQuery/ViewQuery "full") by table.
// Expected columns: table_name, col_name, col_type, not_null, col_default, col_description, tbl_comment [, view_source]
func groupTableColumns(rows [][]any) (map[string]map[string]any, []string) {
	tableMap := map[string]map[string]any{}
	tableOrder := []string{}

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		tableName := valStr(row[0])
		if _, exists := tableMap[tableName]; !exists {
			tableOrder = append(tableOrder, tableName)
			t := map[string]any{
				"name":    tableName,
				"columns": []map[string]any{},
			}
			if len(row) > 6 {
				if comment := valStr(row[6]); comment != "" {
					t["comment"] = comment
				}
			}
			if len(row) > 7 {
				if src := valStr(row[7]); src != "" {
					t["view_source"] = src
				}
			}
			tableMap[tableName] = t
		}

		col := map[string]any{}
		if len(row) > 1 && row[1] != nil {
			col["name"] = valStr(row[1])
		}
		if len(row) > 2 && row[2] != nil {
			if t := valStr(row[2]); t != "" {
				col["type"] = strings.ToLower(t)
			}
		}
		nullable := true
		if len(row) > 3 && row[3] != nil {
			if n, ok := toInt(row[3]); ok {
				nullable = n == 0
			}
		}
		col["nullable"] = nullable
		if len(row) > 4 && row[4] != nil {
			if d := valStr(row[4]); d != "" {
				col["default"] = d
			}
		}
		if len(row) > 5 && row[5] != nil {
			if d := valStr(row[5]); d != "" {
				col["description"] = d
			}
		}

		existing := tableMap[tableName]["columns"].([]map[string]any)
		tableMap[tableName]["columns"] = append(existing, col)
	}
	return tableMap, tableOrder
}

// mergeIndexes adds index info to the table map in-place.
// rows: table_name, idx_name, is_unique, constraint_type, col_name, col_pos
func mergeIndexes(tableMap map[string]map[string]any, rows [][]any) {
	type key struct{ table, index string }
	idxMap := map[key]map[string]any{}
	idxOrder := map[string][]string{}
	seen := map[key]bool{}

	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		tableName := valStr(row[0])
		if _, exists := tableMap[tableName]; !exists {
			continue
		}
		idxName := valStr(row[1])
		k := key{tableName, idxName}
		if !seen[k] {
			seen[k] = true
			isUnique := false
			if len(row) > 2 {
				if n, ok := toInt(row[2]); ok {
					isUnique = n == 1
				}
			}
			isPrimary := false
			if len(row) > 3 && row[3] != nil {
				isPrimary = strings.TrimSpace(valStr(row[3])) == "PRIMARY KEY"
			}
			idxMap[k] = map[string]any{
				"name":    idxName,
				"columns": []string{},
				"unique":  isUnique,
				"primary": isPrimary,
			}
			idxOrder[tableName] = append(idxOrder[tableName], idxName)
		}
		colName := valStr(row[4])
		idxMap[k]["columns"] = append(idxMap[k]["columns"].([]string), colName)
	}

	for tableName, names := range idxOrder {
		if _, exists := tableMap[tableName]; !exists {
			continue
		}
		indexes := make([]map[string]any, 0, len(names))
		for _, name := range names {
			indexes = append(indexes, idxMap[key{tableName, name}])
		}
		tableMap[tableName]["indexes"] = indexes
	}
}

func valStr(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s)
	case []byte:
		return strings.TrimSpace(string(s))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	}
	return 0, false
}
