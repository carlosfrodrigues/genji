package parser

import "github.com/genjidb/genji/sql/query/expr"

// Options of the SQL parser.
type Options struct {
	// A map of builtin SQL functions.
	Functions expr.Functions
}

func defaultOptions() *Options {
	return &Options{
		Functions: expr.NewFunctions(),
	}
}
