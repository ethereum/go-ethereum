FROM postgres:10.4-alpine

# @NOTE:Shyft - need to modify scripts to check db/table existence
WORKDIR /
COPY ./shyft-cli/postgres_setup/docker_init_user_db.sh /docker-entrypoint-initdb.d/docker_init_user_db.sh
RUN cd /docker-entrypoint-initdb.d && ls -a
VOLUME ["/var/lib/postgresql"]
EXPOSE 5432