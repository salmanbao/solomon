# GORM Postgres Policy

## Goal
Use GORM APIs for PostgreSQL data access in Solomon module/repository code to reduce SQL-injection risk and standardize persistence patterns.

## Required Patterns
- Use `db.WithContext(ctx)` for all DB operations.
- Use builder-style filtering:
  - `Where("field = ?", value)`
  - `Where("field IN ?", values)`
- Use transactions via `db.Transaction(func(tx *gorm.DB) error { ... })`.
- Keep DTO/domain mapping separate from persistence model structs.

## Forbidden Patterns
- `database/sql` query execution in module repository code.
- Raw query construction via string concatenation or `fmt.Sprintf`.
- Embedding user input into SQL literals.

## Safe Query Examples
```go
var clip ClipModel
err := db.WithContext(ctx).
    Where("clip_id = ?", clipID).
    First(&clip).Error
```

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&claim).Error; err != nil {
        return err
    }
    return tx.Create(&outbox).Error
})
```
