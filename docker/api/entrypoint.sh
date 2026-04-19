#!/bin/sh
set -e

echo "Running migrations..."
node dist/infrastructure/persistence/migrate.js

echo "Starting server..."
exec node dist/main.js
