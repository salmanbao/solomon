# GORM Remediation Checklist

1. Identify direct SQL usage in Go files.
2. Replace `sql.DB` query calls with `*gorm.DB` calls.
3. Convert SQL strings into GORM clauses (`Where`, `Order`, `Limit`, `Offset`).
4. Ensure all dynamic predicates are bound parameters.
5. Keep transaction semantics intact when replacing SQL transactions.
6. Preserve error mapping (`not found`, `conflict`, `validation`).
7. Update tests for query behavior parity.
8. Run:
   - `python .agents/skills/solomon-gorm-postgres-enforcer/scripts/check_gorm_postgres.py .`
   - `go test ./...`
