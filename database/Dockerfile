FROM postgres:17.0 AS base

# Install pg_cron
RUN apt-get update && \
    apt-get install curl ca-certificates gnupg lsb-release -y && \
    curl https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
    echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    apt-get update && \
    apt-get install -y postgis postgresql-17-postgis-3 postgresql-17-cron && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy init scripts
COPY ./config/ /docker-entrypoint-initdb.d/

USER postgres
