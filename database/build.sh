set -e -x

# Build base Docker image
docker build -t mds-us-database -f Dockerfile .

# Remove old containers and start a fresh one
docker rm -f mds-us-database || true
docker run -d --name mds-us-database -p 5432:5432 mds-us-database

until pg_isready -h localhost -p 5432; do
  echo "Waiting for database to start..."
  sleep 2
done

echo "Database is ready. Bootstrapping..."

# Run bootstrap scripts
./bootstrap.sh

docker stop -t 120 mds-us-database

docker commit mds-us-database mds-us-database
docker tag mds-us-database ghcr.io/metdatasystem/us/database:latest

