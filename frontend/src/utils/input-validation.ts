// Data type validation utilities for form inputs

// Type sets for validation
const intTypes = new Set([
    "INTEGER", "SMALLINT", "BIGINT", "INT", "INT8", "INT4", "INT2", "SERIAL", "BIGSERIAL", "MEDIUMINT", "TINYINT"
]);

const uintTypes = new Set([
    "UINT8", "UINT4", "UINT2", "UINT", "UINTEGER", "UBIGINT", "USMALLINT"
]);

const floatTypes = new Set([
    "REAL", "NUMERIC", "DOUBLE PRECISION", "FLOAT", "NUMBER", "DOUBLE", "DECIMAL", 
    "FLOAT4", "FLOAT8", "MONEY"
]);

const boolTypes = new Set([
    "BOOLEAN", "BIT", "BOOL"
]);

const dateTypes = new Set([
    "DATE"
]);

const dateTimeTypes = new Set([
    "DATETIME", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE", 
    "DATETIME2", "SMALLDATETIME", "TIMETZ", "TIMESTAMPTZ"
]);

const uuidTypes = new Set([
    "UUID"
]);

// Get the HTML input type based on column type
export function getInputTypeForColumnType(columnType: string | undefined): string {
    if (!columnType) return "text";
    
    const type = columnType.toUpperCase();
    
    // For numeric types
    if (intTypes.has(type) || uintTypes.has(type) || floatTypes.has(type)) {
        return "text"; // We'll use text with pattern validation for better control
    }
    
    // For date/datetime types
    if (dateTypes.has(type)) {
        return "date";
    }
    if (dateTimeTypes.has(type)) {
        return "datetime-local";
    }
    
    return "text";
}

// Get input pattern for validation
export function getInputPatternForColumnType(columnType: string | undefined): string | undefined {
    if (!columnType) return undefined;
    
    const type = columnType.toUpperCase();
    
    // Integer types - allow negative and positive integers
    if (intTypes.has(type)) {
        return "^-?\\d+$";
    }
    
    // Unsigned integer types - only positive integers
    if (uintTypes.has(type)) {
        return "^\\d+$";
    }
    
    // Float types - allow decimal numbers
    if (floatTypes.has(type)) {
        return "^-?\\d*\\.?\\d+$";
    }
    
    // UUID pattern
    if (uuidTypes.has(type)) {
        return "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$|^gen_random_uuid\\(\\)$";
    }
    
    return undefined;
}

// Validate input value based on column type
export function validateInputForColumnType(value: string, columnType: string | undefined): boolean {
    if (!columnType || value === "") return true; // Allow empty values
    
    const type = columnType.toUpperCase();
    
    // Integer validation
    if (intTypes.has(type)) {
        return /^-?\d+$/.test(value);
    }
    
    // Unsigned integer validation
    if (uintTypes.has(type)) {
        return /^\d+$/.test(value);
    }
    
    // Float validation
    if (floatTypes.has(type)) {
        return /^-?\d*\.?\d+$/.test(value);
    }
    
    // Boolean validation
    if (boolTypes.has(type)) {
        const lowerValue = value.toLowerCase();
        return ["true", "false", "1", "0", "yes", "no", "t", "f"].includes(lowerValue);
    }
    
    // UUID validation - allow gen_random_uuid() function or actual UUID
    if (uuidTypes.has(type)) {
        return /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/.test(value) ||
               value === "gen_random_uuid()";
    }
    
    return true; // Allow any value for other types
}

// Get placeholder text for input based on column type
export function getPlaceholderForColumnType(columnType: string | undefined): string {
    if (!columnType) return "";
    
    const type = columnType.toUpperCase();
    
    if (intTypes.has(type)) return "Enter integer (e.g., 123)";
    if (uintTypes.has(type)) return "Enter positive integer (e.g., 123)";
    if (floatTypes.has(type)) return "Enter number (e.g., 123.45)";
    if (boolTypes.has(type)) return "Enter true/false";
    if (dateTypes.has(type)) return "Select date";
    if (dateTimeTypes.has(type)) return "Select date and time";
    if (uuidTypes.has(type)) return "Enter UUID or gen_random_uuid()";
    
    return "";
}

// Filter keyboard input based on column type
export function filterKeyboardInput(event: React.KeyboardEvent<HTMLInputElement>, columnType: string | undefined): boolean {
    if (!columnType) return true; // Allow all input if no type specified
    
    const type = columnType.toUpperCase();
    const key = event.key;
    const currentValue = event.currentTarget.value;
    const selectionStart = event.currentTarget.selectionStart || 0;
    
    // Always allow control keys
    if (key === "Backspace" || key === "Delete" || key === "Tab" || 
        key === "Enter" || key === "ArrowLeft" || key === "ArrowRight" ||
        key === "Home" || key === "End" || event.ctrlKey || event.metaKey) {
        return true;
    }
    
    // For numeric types
    if (intTypes.has(type) || uintTypes.has(type) || floatTypes.has(type)) {
        // Allow numbers
        if (/\d/.test(key)) return true;
        
        // Allow minus sign at the beginning for signed types
        if (key === "-" && !uintTypes.has(type) && selectionStart === 0 && !currentValue.includes("-")) {
            return true;
        }
        
        // Allow decimal point for float types
        if (key === "." && floatTypes.has(type) && !currentValue.includes(".")) {
            return true;
        }
        
        // Block other characters
        event.preventDefault();
        return false;
    }
    
    return true; // Allow all input for other types
}

// Format value based on column type (for display purposes)
export function formatValueForColumnType(value: string, columnType: string | undefined): string {
    if (!value || !columnType) return value;
    
    const type = columnType.toUpperCase();
    
    // Format boolean values
    if (boolTypes.has(type)) {
        const lowerValue = value.toLowerCase();
        if (["true", "1", "yes", "t"].includes(lowerValue)) return "true";
        if (["false", "0", "no", "f"].includes(lowerValue)) return "false";
    }
    
    return value;
}