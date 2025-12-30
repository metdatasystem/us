psql -h localhost -U postgres -f "./init.sql" || exit 2

FILES=("public" "postgis" "awips" "vtec" "warnings" "mcd")

# Load tables
for sql_file in ${FILES[@]}; do        
    echo "Running $sql_file.sql"
    psql -h localhost -U postgres -d mds -f "./schemas/$sql_file.sql" || exit 2
done

for sql_file in states offices vtec; do
    echo "Running $sql_file"
    psql -h localhost -U postgres -d mds -f "./data/$sql_file.sql" || exit 2
done