import React, { useState, useCallback } from 'react';
import { notify } from '../store/function';
import { ImportModal } from './ImportModal';

interface CSVOperationsProps {
  schema: string;
  storageUnit: string;
  selectedRowCount?: number;
  selectedRows?: Record<string, any>[];
  onImportComplete?: () => void;
}

export const CSVOperations: React.FC<CSVOperationsProps> = ({
  schema,
  storageUnit,
  selectedRowCount = 0,
  selectedRows,
  onImportComplete
}) => {
  const [showImportModal, setShowImportModal] = useState(false);
  const [showExportConfirm, setShowExportConfirm] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [delimiter, setDelimiter] = useState(',');
  const [exportFormat, setExportFormat] = useState<'csv' | 'excel'>('csv');

  // Export handler using HTTP streaming endpoint
  const handleExport = useCallback(async (e?: React.MouseEvent) => {
    if (e) {
      e.preventDefault();
      e.stopPropagation();
    }
    setExporting(true);
    try {
      // Use fetch API to download CSV via streaming endpoint
      const response = await fetch('/api/export', {
        method: 'POST',
        credentials: 'include', // Include auth cookies
        headers: {
          'Accept': 'text/csv',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          schema,
          storageUnit,
          delimiter,
          format: exportFormat,
          ...(selectedRows && selectedRows.length > 0 ? { selectedRows } : {}),
        }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || `Export failed: ${response.statusText}`);
      }

      // Get filename from Content-Disposition header
      const contentDisposition = response.headers.get('Content-Disposition');
      let filename = `${schema}_${storageUnit}.${exportFormat === 'excel' ? 'xlsx' : 'csv'}`;
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename="(.+)"/);
        if (filenameMatch) {
          filename = filenameMatch[1];
        }
      }

      // Stream the response directly to a download
      const blob = await response.blob();
      
      // Create a download link
      const downloadUrl = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = downloadUrl;
      link.download = filename;
      link.style.display = 'none';
      document.body.appendChild(link);
      
      // Trigger download
      link.click();
      
      // Cleanup
      setTimeout(() => {
        document.body.removeChild(link);
        window.URL.revokeObjectURL(downloadUrl);
      }, 100);

      notify('Export completed successfully', 'success');
    } catch (error: any) {
      console.error('Export failed:', error);
      notify(`Export failed: ${error.message}`, 'error');
    } finally {
      setExporting(false);
      setShowExportConfirm(false);
    }
  }, [schema, storageUnit, delimiter]);

  return (
    <>
      {/* Export Button */}
      <button
        type="button"
        onClick={() => setShowExportConfirm(true)}
        disabled={exporting}
        className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:bg-gray-300"
      >
        {exporting ? 'Exporting...' : `Export ${selectedRowCount > 0 ? `${selectedRowCount} selected` : 'all'}`}
      </button>

      {/* Import Button */}
      <button
        onClick={() => setShowImportModal(true)}
        className="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"
      >
        Import CSV
      </button>

      {/* Export Confirmation Dialog */}
      {showExportConfirm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md">
            <h3 className="text-lg font-semibold mb-4">Export Data</h3>
            <p className="mb-4">
              Export {selectedRowCount > 0 ? `${selectedRowCount} selected rows` : 'all rows'} 
              from {schema}.{storageUnit}?
            </p>
            <div className="mb-4">
              <label className="block text-sm font-medium mb-2">
                Format
              </label>
              <select
                value={exportFormat}
                onChange={(e) => setExportFormat(e.target.value as 'csv' | 'excel')}
                className="w-full p-2 border rounded mb-3"
              >
                <option value="csv">CSV - Comma Separated Values</option>
                <option value="excel">Excel - XLSX Format</option>
              </select>
              
              {exportFormat === 'csv' && (
                <>
                  <label className="block text-sm font-medium mb-2">
                    Delimiter
                  </label>
                  <select
                    value={delimiter}
                    onChange={(e) => setDelimiter(e.target.value)}
                    className="w-full p-2 border rounded"
                  >
                    <option value=",">Comma (,) - Standard CSV</option>
                    <option value=";">Semicolon (;) - Excel in some locales</option>
                    <option value="|">Pipe (|) - Less common in data</option>
                    <option value="\t">Tab - TSV format</option>
                  </select>
                  <p className="text-xs text-gray-600 mt-1">
                    Choose a delimiter that doesn't appear in your data
                  </p>
                </>
              )}
            </div>
            <div className="bg-gray-100 p-3 rounded mb-4">
              <p className="text-sm">
                <strong>Encoding:</strong> UTF-8<br/>
                <strong>Headers:</strong> Included with type information
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowExportConfirm(false)}
                className="px-4 py-2 border rounded hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleExport}
                className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
              >
                Export
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Import Modal (uses HTTP endpoint) */}
      <ImportModal
        isOpen={showImportModal}
        onClose={() => setShowImportModal(false)}
        schema={schema}
        storageUnit={storageUnit}
        onImportComplete={() => {
          setShowImportModal(false);
          onImportComplete?.();
        }}
      />
    </>
  );
};