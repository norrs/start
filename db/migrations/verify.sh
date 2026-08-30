#!/usr/bin/env bash
set -euo pipefail
grep -q -- '-- +goose Up' db/migrations/00001_initial.sql
grep -q -- '-- +goose Down' db/migrations/00001_initial.sql
