package postgres

import (
	"context"
	"fmt"

	"github.com/ciphersync/core/domain"
	"github.com/jackc/pgx/v5"
)

type Introspector struct {
	conn *pgx.Conn
}

func NewIntrospector(conn *pgx.Conn) *Introspector {
	return &Introspector{conn: conn}
}

func (i *Introspector) Introspect(ctx context.Context) ([]*domain.Table, error) {
	tablesMap := make(map[string]*domain.Table)

	// 1. Fetch Tables and Columns
	colsQuery := `
		SELECT c.relname, n.nspname, a.attname, format_type(a.atttypid, a.atttypmod), not a.attnotnull
		FROM pg_attribute a
		JOIN pg_class c ON a.attrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE a.attnum > 0 AND not a.attisdropped AND n.nspname NOT IN ('information_schema', 'pg_catalog') AND c.relkind = 'r'
		ORDER BY n.nspname, c.relname, a.attnum;
	`
	rows, err := i.conn.Query(ctx, colsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, schemaName, colName, dataType string
		var isNullable bool
		if err := rows.Scan(&tableName, &schemaName, &colName, &dataType, &isNullable); err != nil {
			return nil, err
		}

		key := schemaName + "." + tableName
		if _, exists := tablesMap[key]; !exists {
			tablesMap[key] = &domain.Table{
				Schema:      schemaName,
				Name:        tableName,
				Columns:     []domain.Column{},
				PrimaryKeys: []string{},
				ForeignKeys: []domain.ForeignKey{},
			}
		}

		tablesMap[key].Columns = append(tablesMap[key].Columns, domain.Column{
			Name:       colName,
			DataType:   dataType,
			IsNullable: isNullable,
		})
	}

	// 2. Fetch Primary Keys
	pkQuery := `
		SELECT n.nspname, c.relname, a.attname
		FROM pg_index idx
		JOIN pg_class c ON c.oid = idx.indrelid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(idx.indkey)
		WHERE idx.indisprimary AND n.nspname NOT IN ('information_schema', 'pg_catalog');
	`
	pkRows, err := i.conn.Query(ctx, pkQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query primary keys: %w", err)
	}
	defer pkRows.Close()

	for pkRows.Next() {
		var schemaName, tableName, colName string
		if err := pkRows.Scan(&schemaName, &tableName, &colName); err != nil {
			return nil, err
		}
		key := schemaName + "." + tableName
		if tbl, exists := tablesMap[key]; exists {
			tbl.PrimaryKeys = append(tbl.PrimaryKeys, colName)
		}
	}

	// 3. Fetch Foreign Keys (handles composite keys via array aggregation and ordinality)
	fkQuery := `
		SELECT 
			con.conname,
			src_ns.nspname, src_cl.relname, tgt_ns.nspname, tgt_cl.relname,
			(SELECT array_agg(a.attname ORDER BY x.ord) FROM unnest(con.conkey) WITH ORDINALITY x(attnum, ord) JOIN pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = x.attnum),
			(SELECT array_agg(a.attname ORDER BY x.ord) FROM unnest(con.confkey) WITH ORDINALITY x(attnum, ord) JOIN pg_attribute a ON a.attrelid = con.confrelid AND a.attnum = x.attnum)
		FROM pg_constraint con
		JOIN pg_class src_cl ON con.conrelid = src_cl.oid
		JOIN pg_namespace src_ns ON src_cl.relnamespace = src_ns.oid
		JOIN pg_class tgt_cl ON con.confrelid = tgt_cl.oid
		JOIN pg_namespace tgt_ns ON tgt_cl.relnamespace = tgt_ns.oid
		WHERE con.contype = 'f' AND src_ns.nspname NOT IN ('information_schema', 'pg_catalog');
	`
	fkRows, err := i.conn.Query(ctx, fkQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer fkRows.Close()

	for fkRows.Next() {
		var conName, srcSchema, srcTable, tgtSchema, tgtTable string
		var srcCols, tgtCols []string

		if err := fkRows.Scan(&conName, &srcSchema, &srcTable, &tgtSchema, &tgtTable, &srcCols, &tgtCols); err != nil {
			return nil, err
		}

		key := srcSchema + "." + srcTable
		if tbl, exists := tablesMap[key]; exists {
			// Prepend target schema to target table name if they differ or for explicit mapping
			fullTarget := tgtTable
			if tgtSchema != "public" {
				fullTarget = tgtSchema + "." + tgtTable
			}

			tbl.ForeignKeys = append(tbl.ForeignKeys, domain.ForeignKey{
				ConstraintName: conName,
				SourceColumns:  srcCols,
				TargetTable:    fullTarget,
				TargetColumns:  tgtCols,
			})
		}
	}

	// Flatten map to slice
	result := make([]*domain.Table, 0)
	for _, tbl := range tablesMap {
		result = append(result, tbl)
	}

	return result, nil
}
