# Copyright 2025 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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