FROM postgis/postgis:17-3.5 AS base

# Install pg_cron
RUN apt-get update && \
    apt-get install -y postgresql-17-cron && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy init scripts
COPY ./config/ /docker-entrypoint-initdb.d/
COPY ./data/ /docker-entrypoint-initdb.d/data
COPY ./schemas/ /docker-entrypoint-initdb.d/schemas
COPY ./migrations/ /docker-entrypoint-initdb.d/migrations
