// Package repository contains IAM persistence implementations.
//
// Repository row types are persistence-only, use sqlx db tags for column
// mapping, and may use PostgreSQL-specific persistence types when they provide
// concrete value. Domain models do not contain database mapping tags or SQL
// storage concerns.
package repository
