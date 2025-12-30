#!/bin/bash
set -e

echo "Initializing database..."

psql -U postgres -f "/docker-entrypoint-initdb.d/init.sql"

FILES=("public" "postgis" "awips" "vtec" "warnings" "mcd")

# Load tables
for sql_file in ${FILES[@]}; do        
    echo "Running $sql_file.sql"
    psql -U postgres -d mds -f "./schemas/$sql_file.sql"
done
    
psql -U postgres -c "ALTER DATABASE mds SET search_path = public, postgis, awips, vtec, warnings, mcd"

# Load data
for sql_file in states offices vtec cron; do
    echo "Running $sql_file"
    psql -U postgres -d mds -f "/docker-entrypoint-initdb.d/data/$sql_file.sql"
done


echo "Done."