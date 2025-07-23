import { DatabaseType } from "../generated/graphql";
import { Icons } from "../components/icons";
import { IDropdownItem } from "../components/dropdown";
import { EEDatabaseType } from "./ee-types";

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

// This will be populated if EE is loaded
let eeDatabaseTypes: IDropdownItem<Record<string, string>>[] = [];

// Load EE database types if in EE mode
if (import.meta.env.VITE_BUILD_EDITION === 'ee') {
    // Load EE config asynchronously
    Promise.all([
        import('@ee/config'),
        import('@ee/icons')
    ]).then(([eeConfig, eeIcons]) => {
        if (eeConfig?.eeDatabaseTypes && eeIcons?.EEIcons?.Logos) {
            // First merge the icons
            Object.assign(Icons.Logos, eeIcons.EEIcons.Logos);
            
            // Then map EE database types to the correct format with resolved icons
            eeDatabaseTypes = eeConfig.eeDatabaseTypes.map(dbType => ({
                id: dbType.id,
                label: dbType.label,
                icon: Icons.Logos[dbType.iconName as keyof typeof Icons.Logos],
                extra: dbType.extra,
            }));
        }
    }).catch(() => {
        console.warn('Could not load EE database types');
    });
}

// Get all database types based on whether EE is enabled
export const getDatabaseTypeDropdownItems = (): IDropdownItem<Record<string, string>>[] => {
    const isEE = import.meta.env.VITE_BUILD_EDITION === 'ee';
    
    if (isEE && eeDatabaseTypes.length > 0) {
        return [...baseDatabaseTypes, ...eeDatabaseTypes];
    }
    
    return baseDatabaseTypes;
};

export const databaseTypeDropdownItems = getDatabaseTypeDropdownItems();