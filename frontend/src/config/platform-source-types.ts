/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * A source type that is only connectable on WhoDB Platform. Shown in the CE
 * connection picker (behind the platform funnel flag) so users searching for
 * one discover it exists, with selection routing to the platform explainer
 * instead of a login form.
 */
export type PlatformSourceType = {
    /** Identifier matching the platform catalog; also the icon registry key. */
    id: string;
    /** Display label shown in the picker. */
    label: string;
};

/**
 * Curated list of recognizable platform-only source types. Deliberately not
 * exhaustive — these are the types CE users most commonly search for.
 */
export const PLATFORM_SOURCE_TYPES: PlatformSourceType[] = [
    { id: "MSSQL", label: "Microsoft SQL Server" },
    { id: "Oracle", label: "Oracle" },
    { id: "Snowflake", label: "Snowflake" },
    { id: "BigQuery", label: "BigQuery" },
    { id: "DynamoDB", label: "DynamoDB" },
    { id: "S3", label: "Amazon S3" },
    { id: "Redshift", label: "Amazon Redshift" },
    { id: "Athena", label: "Amazon Athena" },
    { id: "Cassandra", label: "Cassandra" },
    { id: "Databricks", label: "Databricks" },
    { id: "Spanner", label: "Cloud Spanner" },
    { id: "Neo4j", label: "Neo4j" },
    { id: "Trino", label: "Trino" },
    { id: "Azure Cosmos DB", label: "Azure Cosmos DB" },
];

/**
 * Finds a platform-only source type by picker value.
 */
export const findPlatformSourceType = (id: string): PlatformSourceType | undefined =>
    PLATFORM_SOURCE_TYPES.find(sourceType => sourceType.id === id);
