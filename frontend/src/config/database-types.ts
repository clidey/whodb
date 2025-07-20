import { DatabaseType } from "../generated/graphql";
import { Icons } from "../components/icons";
import { IDropdownItem } from "../components/dropdown";

export const baseDatabaseTypes: IDropdownItem<Record<string, string>>[] = [
    {
        id: "Postgres",
        label: "Postgres",
        icon: Icons.Logos.Postgres,
        extra: {"Port": "5432"},
    },
    {
        id: "MySQL",
        label: "MySQL",
        icon: Icons.Logos.MySQL,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "Local", "Allow clear text passwords": "0"},
    },
    {
        id: "MariaDB",
        label: "MariaDB",
        icon: Icons.Logos.MariaDB,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "Local", "Allow clear text passwords": "0"},
    },
    {
        id: "Sqlite3",
        label: "Sqlite3",
        icon: Icons.Logos.Sqlite3,
        extra: {},
    },
    {
        id: "MongoDB",
        label: "MongoDB",
        icon: Icons.Logos.MongoDB,
        extra: {"Port": "27017", "URL Params": "?", "DNS Enabled": "false"},
    },
    {
        id: "Redis",
        label: "Redis",
        icon: Icons.Logos.Redis,
        extra: {"Port": "6379"},
    },
    {
        id: "ElasticSearch",
        label: "ElasticSearch",
        icon: Icons.Logos.ElasticSearch,
        extra: {"Port": "9200", "SSL Mode": "disable"},
    },
    {
        id: "ClickHouse",
        label: "ClickHouse",
        icon: Icons.Logos.ClickHouse,
        extra: {
            "Port": "9000",
            "SSL mode": "disable",
            "HTTP Protocol": "disable",
            "Readonly": "disable",
            "Debug": "disable"
        }
    },
];

export const eeDatabaseTypes: IDropdownItem<Record<string, string>>[] = [
    {
        id: "MSSQL",
        label: "Microsoft SQL Server",
        icon: Icons.Logos.MSSQL,
        extra: {"Port": "1433"},
    },
    {
        id: "Oracle",
        label: "Oracle",
        icon: Icons.Logos.Oracle,
        extra: {"Port": "1521"},
    },
    {
        id: "DynamoDB",
        label: "AWS DynamoDB",
        icon: Icons.Logos.DynamoDB,
        extra: {"Region": "us-east-1"},
    },
];

// Get all database types based on whether EE is enabled
export const getDatabaseTypeDropdownItems = (): IDropdownItem<Record<string, string>>[] => {
    const isEE = process.env.BUILD_EDITION === 'ee';
    
    if (isEE) {
        return [...baseDatabaseTypes, ...eeDatabaseTypes];
    }
    
    return baseDatabaseTypes;
};

export const databaseTypeDropdownItems = getDatabaseTypeDropdownItems();