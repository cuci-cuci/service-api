#!/bin/sh
set -e
echo "=== Container Starting ==="
echo "Architecture: $(uname -m)"
echo "DATABASE_URL length: ${#DATABASE_URL}"
echo "JWT_SECRET length: ${#JWT_SECRET}"
echo "PORT: $PORT"
echo "=== Starting server ==="
exec ./server
