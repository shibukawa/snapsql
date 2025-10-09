#!/bin/sh
go run ./cmd/notification -dsn postgres://notification:notification@localhost:5432/notification?sslmode=disable
