package psql

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

func TestSelect(t *testing.T) {
	var qm = SelectQM{}
	var examples = d.Testcases{
		"simple select": {
			ExpectedQuery: "SELECT id, name FROM users WHERE (id IN ($1, $2, $3))",
			ExpectedArgs:  []any{100, 200, 300},
			Query: Select(
				qm.Select("id", "name"),
				qm.From("users"),
				qm.Where(qm.X("id").IN(qm.Arg(100, 200, 300))),
			),
		},
		"with rows from": {
			Query: Select(
				qm.From(
					expr.Func(
						"json_to_recordset",
						qm.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
					).Col("a", "INTEGER").Col("b", "TEXT").ToMod(),
					expr.Func("generate_series", 1, 3).ToMod(),
					qm.As("x", "p", "q", "s"),
				),
				qm.OrderBy("p"),
			),
			ExpectedQuery: ` SELECT *
				FROM ROWS FROM
					(
						json_to_recordset($1)
							AS (a INTEGER, b TEXT),
						generate_series(1, 3)
					) AS "x" ("p", "q", "s")
				ORDER BY p`,
			ExpectedArgs: []any{`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`},
		},
		"with sub-select": {
			ExpectedQuery: `SELECT status, avg(difference)
					FROM (
						SELECT
						status, ((
							LEAD(created_date, 1, NOW())
							OVER(PARTITION BY presale_id ORDER BY created_date)
						) - created_date) AS "difference"
						FROM presales_presalestatus
					) AS "differnce_by_status"
					WHERE (status IN ('A', 'B', 'C'))
					GROUP BY status`,
			Query: Select(
				qm.Select("status", expr.Func("avg", "difference")),
				qm.From(
					Select(
						qm.Select(
							"status",
							qm.OVER(
								expr.Func("LEAD", "created_date", 1, expr.Func("NOW")),
								expr.Window("").PartitionBy("presale_id").OrderBy("created_date"),
							).MINUS("created_date").AS("difference"),
						),
						qm.From("presales_presalestatus"),
					),
					qm.As("differnce_by_status"),
				),
				qm.Where(qm.X("status").IN(expr.S("A"), expr.S("B"), expr.S("C"))),
				qm.GroupBy("status"),
			),
		},
	}

	for name, tc := range examples {
		t.Run(name, func(t *testing.T) {
			sql, args, err := query.Build(tc.Query)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if diff := d.QueryDiff(tc.ExpectedQuery, sql); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
			if diff := d.ArgsDiff(tc.ExpectedArgs, args); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}
