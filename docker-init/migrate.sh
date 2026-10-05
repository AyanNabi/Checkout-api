#!/bin/sh

set -e

for file in /migrations/*.up.sql
do
    echo "Running migration: $file"
    psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f "$file"
done