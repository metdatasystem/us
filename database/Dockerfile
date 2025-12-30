FROM postgis/postgis:18-3.6

RUN mkdir /opt/pgdata && chown postgres:postgres /opt/pgdata && chmod 700 /opt/pgdata
ENV PGDATA=/opt/pgdata

# Switch back to postgres user (required by official image)
USER postgres

RUN initdb -D /opt/pgdata --auth-host=trust --auth-local=trust
RUN echo "host all all 172.0.0.0/8 trust" >> /opt/pgdata/pg_hba.conf
