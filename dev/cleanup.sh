docker rm dev-e2e_sqlite3-1
docker rm dev-e2e_postgres-1
docker rm dev-e2e_mysql-1
docker rm dev-e2e_mariadb-1
docker rm dev-e2e_mongo-1
docker rm dev-e2e_clickhouse-1

docker volume rm dev_e2e_sqlite3
docker volume rm dev_e2e_postgres
docker volume rm dev_e2e_mysql
docker volume rm dev_e2e_mariadb
docker volume rm dev_e2e_mongo
docker volume rm dev_e2e_clickhouse