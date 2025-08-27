import { ReactElement } from "react";
import { Icons } from "../components/icons";

// Extended dropdown item type with UI field configuration
export interface IDatabaseDropdownItem {
    id: string;
    label: string;
    icon: ReactElement;
    extra: Record<string, string>;
    // UI field configuration
    fields?: {
        hostname?: boolean;
        username?: boolean;
        password?: boolean;
        database?: boolean;
    };
    // Valid operators for this database type
    operators?: string[];
    // Valid data types for creating tables/collections
    dataTypes?: string[];
    // Whether this database supports field modifiers (primary, nullable)
    supportsModifiers?: boolean;
    // Whether this database supports scratchpad/raw query execution
    supportsScratchpad?: boolean;
    // Whether this database supports schemas
    supportsSchema?: boolean;
}

export const baseDatabaseTypes: IDatabaseDropdownItem[] = [
    {
        id: "Postgres",
        label: "Postgres",
        icon: Icons.Logos.Postgres,
        extra: {"Port": "5432"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        supportsScratchpad: true,
        supportsSchema: true,
    },
    {
        id: "MySQL",
        label: "MySQL",
        icon: Icons.Logos.MySQL,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "Local", "Allow clear text passwords": "0"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        supportsScratchpad: true,
        supportsSchema: true,
    },
    {
        id: "MariaDB",
        label: "MariaDB",
        icon: Icons.Logos.MariaDB,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "Local", "Allow clear text passwords": "0"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        supportsScratchpad: true,
        supportsSchema: true,
    },
    {
        id: "Sqlite3",
        label: "Sqlite3",
        icon: Icons.Logos.Sqlite3,
        extra: {},
        fields: {
            database: true,  // SQLite only needs database field
        },
        supportsScratchpad: true,
        supportsSchema: false,  // SQLite doesn't support schemas
    },
    {
        id: "MongoDB",
        label: "MongoDB",
        icon: Icons.Logos.MongoDB,
        extra: {"Port": "27017", "URL Params": "?", "DNS Enabled": "false"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            // database: false - MongoDB doesn't use database field
        },
        supportsScratchpad: false,  // MongoDB doesn't support SQL scratchpad
        supportsSchema: false,  // MongoDB doesn't have traditional schemas
    },
    {
        id: "Redis",
        label: "Redis",
        icon: Icons.Logos.Redis,
        extra: {"Port": "6379"},
        fields: {
            hostname: true,
            // username: false - Redis doesn't use username
            password: true,
            // database: false - Redis doesn't use database field
        },
        supportsScratchpad: false,  // Redis doesn't support SQL scratchpad
        supportsSchema: false,  // Redis doesn't have schemas
    },
    {
        id: "ElasticSearch",
        label: "ElasticSearch",
        icon: Icons.Logos.ElasticSearch,
        extra: {"Port": "9200", "SSL Mode": "disable"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            // database: false - ElasticSearch doesn't use database field
        },
        supportsScratchpad: false,  // ElasticSearch doesn't support SQL scratchpad
        supportsSchema: false,  // ElasticSearch doesn't have schemas
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
        },
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        supportsScratchpad: true,
        supportsSchema: true,
    },
    {
        id: "Memcached",
        label: "Memcached",
        icon: Icons.Logos.Memcached,
        extra: {"Port": "11211"},
        fields: {
            hostname: true,
            // username: false - Memcached doesn't use authentication by default
            // password: false - Memcached doesn't use authentication by default
            // database: false - Memcached doesn't use database field
        },
        supportsScratchpad: true,  // Memcached supports simple commands via scratchpad
        supportsSchema: false,  // Memcached doesn't have schemas
    },
];

// This will be populated if EE is loaded
let eeDatabaseTypes: IDatabaseDropdownItem[] = [];
let eeLoadPromise: Promise<void> | null = null;

// Load EE database types if in EE mode
if (import.meta.env.VITE_BUILD_EDITION === 'ee') {
    // Store the promise so we can await it later
    eeLoadPromise = Promise.all([
        import('@ee/config.tsx'),
        import('@ee/icons')
    ]).then(([eeConfig, eeIcons]) => {
        console.log('Loading EE config:', eeConfig);
        console.log('Loading EE icons:', eeIcons);
        
        if (eeConfig?.eeDatabaseTypes && eeIcons?.EEIcons?.Logos) {
            // First merge the icons
            Object.assign(Icons.Logos, eeIcons.EEIcons.Logos);
            
            // Then map EE database types to the correct format with resolved icons
            // @ts-ignore - TODO: fix this
            eeDatabaseTypes = eeConfig.eeDatabaseTypes.map(dbType => ({
                id: dbType.id,
                label: dbType.label,
                icon: Icons.Logos[dbType.iconName as keyof typeof Icons.Logos],
                extra: dbType.extra,
                fields: dbType.fields,
                operators: dbType.operators,
                dataTypes: dbType.dataTypes,
                supportsModifiers: dbType.supportsModifiers,
                supportsScratchpad: dbType.supportsScratchpad,
                supportsSchema: dbType.supportsSchema,
            }));
            
            console.log('EE database types loaded:', eeDatabaseTypes);
        } else {
            console.warn('EE modules loaded but missing expected exports', {
                hasDatabaseTypes: !!eeConfig?.eeDatabaseTypes,
                hasIcons: !!eeIcons?.EEIcons?.Logos
            });
        }
    }).catch((error) => {
        console.error('Could not load EE database types:', error);
    });
}

// Get all database types - now returns a promise if EE is loading
export const getDatabaseTypeDropdownItems = async (): Promise<IDatabaseDropdownItem[]> => {
    const isEE = import.meta.env.VITE_BUILD_EDITION === 'ee';
    
    if (isEE && eeLoadPromise) {
        // Wait for EE to load
        await eeLoadPromise;
        
        if (eeDatabaseTypes.length > 0) {
            return [...baseDatabaseTypes, ...eeDatabaseTypes];
        }
    }
    
    return baseDatabaseTypes;
};

// For backward compatibility, provide a synchronous version that only returns base types initially
export const getDatabaseTypeDropdownItemsSync = (): IDatabaseDropdownItem[] => {
    const isEE = import.meta.env.VITE_BUILD_EDITION === 'ee';
    
    if (isEE && eeDatabaseTypes.length > 0) {
        return [...baseDatabaseTypes, ...eeDatabaseTypes];
    }
    
    return baseDatabaseTypes;
};

// Export this for components that need immediate access (will be updated when EE loads)
export let databaseTypeDropdownItems = baseDatabaseTypes;

// Update the exported items when EE loads
if (import.meta.env.VITE_BUILD_EDITION === 'ee' && eeLoadPromise) {
    eeLoadPromise.then(() => {
        if (eeDatabaseTypes.length > 0) {
            databaseTypeDropdownItems = [...baseDatabaseTypes, ...eeDatabaseTypes];
        }
    });
}