# Import/Export Feature Implementation

## Overview
This document describes the implementation of CSV import and export functionality for WhoDB database tables.

## Features Implemented

### Export Functionality
- **Backend streaming export**: Exports all data from a table using efficient streaming
- **Selected rows export**: Export only selected rows (client-side)
- **CSV Format**:
  - Pipe (`|`) delimiter to avoid conflicts with commas in data
  - Headers include column name and data type (format: `column_name:data_type`)
  - UTF-8 encoding with BOM for Excel compatibility

### Import Functionality
- **File upload**: Drag-and-drop or click to select CSV files
- **Import modes**: 
  - Append: Add data to existing table
  - Override: Replace all existing data (with transaction rollback on failure)
- **Data validation**: Ensures all required columns are present
- **Transaction support**: All imports are wrapped in transactions for data integrity

## Backend Implementation

### 1. Plugin Interface Extension
- Added `ExportCSV` and `ImportCSV` methods to `PluginFunctions` interface
- Added `ImportMode` enum and `ImportProgress` struct

### 2. Database Implementations
- **GORM-based** (PostgreSQL, MySQL, SQLite): Full streaming export/import with transactions
- **MongoDB**: Document-based export/import with BSON/JSON handling
- **ElasticSearch**: Index-based export/import using scroll API
- **ClickHouse**: Batch-based export/import
- **Redis**: Not supported (no traditional tables)

### 3. GraphQL API
- Query: `ExportCSV(schema: String!, storageUnit: String!): String!`
- Mutation: `ImportCSV(schema: String!, storageUnit: String!, csvData: String!, mode: ImportMode!): ImportProgress!`
- HTTP endpoint: `/api/export-csv` for direct file download

### 4. CSV Utilities
- Common CSV handling in `core/src/common/csv.go`
- Proper escaping and formatting functions
- Header parsing with type information

## Frontend Implementation

### 1. Table Component Updates
- Added Import and Export buttons to table toolbar
- Pass schema and storageUnit props to enable backend operations
- Updated `useExportToCSV` hook to use backend endpoint

### 2. Import Modal
- File selection with validation
- Import mode selection (Append/Override)
- Warning for override mode
- Progress indication during import
- Clear format requirements display

### 3. Export Confirmation
- Shows row count to be exported
- Explains export format
- Confirmation before proceeding

## Usage

### Exporting Data
1. Navigate to any table view
2. Click "Export all" to export entire table
3. Or select specific rows and click "Export X selected"
4. Confirm export in the dialog
5. CSV file will download automatically

### Importing Data
1. Navigate to the table you want to import to
2. Click "Import" button
3. Select CSV file (must match format requirements)
4. Choose import mode (Append or Override)
5. Click Import and wait for completion
6. Table will refresh automatically after successful import

## CSV Format Requirements
- First row must contain headers with format: `column_name:data_type`
- Use pipe (`|`) as delimiter
- All required table columns must be present in CSV
- Empty values are treated as NULL
- Special characters are properly escaped

## Error Handling
- Import failures trigger transaction rollback
- Clear error messages displayed to users
- No data loss on failed operations
- Validation before processing begins

## Security Considerations
- All operations require authentication
- File uploads restricted to CSV format
- Import data validated before processing
- Transactions ensure data integrity