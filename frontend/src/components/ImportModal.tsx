import React, { useState, useCallback } from 'react';
import { notify } from '../store/function';

interface ImportModalProps {
  isOpen: boolean;
  onClose: () => void;
  schema: string;
  storageUnit: string;
  onImportComplete: () => void;
}

export const ImportModal: React.FC<ImportModalProps> = ({
  isOpen,
  onClose,
  schema,
  storageUnit,
  onImportComplete
}) => {
  const [file, setFile] = useState<File | null>(null);
  const [mode, setMode] = useState<'append' | 'override'>('append');
  const [isImporting, setIsImporting] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [delimiter, setDelimiter] = useState(',');
  const [fileFormat, setFileFormat] = useState<'csv' | 'excel' | null>(null);

  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (!selectedFile) return;

    // Client-side validation
    const MAX_SIZE = 50 * 1024 * 1024; // 50MB
    if (selectedFile.size > MAX_SIZE) {
      notify(`File too large. Maximum size is ${MAX_SIZE / (1024 * 1024)}MB`, 'error');
      return;
    }

    const filename = selectedFile.name.toLowerCase();
    const isCSV = selectedFile.type === 'text/csv' || filename.endsWith('.csv');
    const isExcel = filename.endsWith('.xlsx') || filename.endsWith('.xls');

    if (!isCSV && !isExcel) {
      notify('Please select a CSV or Excel file', 'error');
      return;
    }

    setFile(selectedFile);
    setFileFormat(isExcel ? 'excel' : 'csv');
  }, []);

  const handleImport = useCallback(async () => {
    if (!file) {
      notify('Please select a file', 'error');
      return;
    }

    setIsImporting(true);
    setUploadProgress(0);

    const formData = new FormData();
    formData.append('file', file);
    formData.append('schema', schema);
    formData.append('storageUnit', storageUnit);
    formData.append('mode', mode);
    if (fileFormat === 'csv') {
      formData.append('delimiter', delimiter);
    }

    try {
      // Create XMLHttpRequest for progress tracking
      const xhr = new XMLHttpRequest();

      // Track upload progress
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          const percentComplete = (e.loaded / e.total) * 100;
          setUploadProgress(Math.round(percentComplete));
        }
      });

      // Handle completion
      const response = await new Promise<any>((resolve, reject) => {
        xhr.addEventListener('load', () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            try {
              resolve(JSON.parse(xhr.responseText));
            } catch (e) {
              reject(new Error('Invalid response'));
            }
          } else {
            try {
              const error = JSON.parse(xhr.responseText);
              reject(new Error(error.error || `Upload failed: ${xhr.statusText}`));
            } catch (e) {
              reject(new Error(`Upload failed: ${xhr.statusText}`));
            }
          }
        });

        xhr.addEventListener('error', () => {
          reject(new Error('Network error'));
        });

        xhr.addEventListener('abort', () => {
          reject(new Error('Upload cancelled'));
        });

        xhr.open('POST', '/api/import');
        xhr.withCredentials = true; // Include cookies for auth
        xhr.send(formData);
      });

      if (response.status === 'completed') {
        notify(`Successfully imported ${response.processedRows} rows`, 'success');
        onImportComplete();
        onClose();
      } else {
        throw new Error(response.error || 'Import failed');
      }
    } catch (error: any) {
      notify(error.message, 'error');
    } finally {
      setIsImporting(false);
      setUploadProgress(0);
    }
  }, [file, schema, storageUnit, mode, onImportComplete, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full">
        <h2 className="text-xl font-semibold mb-4">Import CSV</h2>
        
        {/* File Input */}
        <div className="mb-4">
          <label className="block text-sm font-medium mb-2">
            Select CSV File
          </label>
          <input
            type="file"
            accept=".csv,text/csv,.xlsx,.xls"
            onChange={handleFileSelect}
            className="w-full p-2 border rounded"
            disabled={isImporting}
          />
          {file && (
            <p className="text-sm text-gray-600 mt-1">
              {file.name} ({(file.size / 1024).toFixed(1)} KB)
            </p>
          )}
        </div>

        {/* Import Mode */}
        <div className="mb-4">
          <label className="block text-sm font-medium mb-2">
            Import Mode
          </label>
          <select
            value={mode}
            onChange={(e) => setMode(e.target.value as 'append' | 'override')}
            className="w-full p-2 border rounded"
            disabled={isImporting}
          >
            <option value="append">Append - Add to existing data</option>
            <option value="override">Override - Replace all data</option>
          </select>
          {mode === 'override' && (
            <p className="text-sm text-red-600 mt-1">
              ⚠️ Warning: This will delete all existing data
            </p>
          )}
        </div>

        {/* Progress Bar */}
        {isImporting && (
          <div className="mb-4">
            <div className="flex justify-between text-sm mb-1">
              <span>Uploading...</span>
              <span>{uploadProgress}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div
                className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                style={{ width: `${uploadProgress}%` }}
              />
            </div>
          </div>
        )}

        {/* Delimiter selection for CSV */}
        {fileFormat === 'csv' && (
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2">
              Delimiter
            </label>
            <select
              value={delimiter}
              onChange={(e) => setDelimiter(e.target.value)}
              className="w-full p-2 border rounded"
              disabled={isImporting}
            >
              <option value=",">Comma (,) - Standard CSV</option>
              <option value=";">Semicolon (;) - Excel in some locales</option>
              <option value="|">Pipe (|) - Less common in data</option>
              <option value="\t">Tab - TSV format</option>
            </select>
            <p className="text-xs text-gray-600 mt-1">
              Choose the delimiter that matches your file
            </p>
          </div>
        )}

        {/* Format Info */}
        <div className="mb-4 p-3 bg-gray-100 rounded text-sm">
          <p className="font-medium mb-1">File Format Requirements:</p>
          <ul className="list-disc list-inside text-gray-600">
            <li>First row must contain column headers with format: column_name:data_type</li>
            {fileFormat === 'csv' && <li>Use selected delimiter to separate values</li>}
            {fileFormat === 'excel' && <li>Data will be imported from the first sheet</li>}
            <li>Maximum file size: 50MB</li>
            <li>Maximum rows: 1,000,000</li>
          </ul>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 border rounded hover:bg-gray-50"
            disabled={isImporting}
          >
            Cancel
          </button>
          <button
            onClick={handleImport}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:bg-gray-300"
            disabled={!file || isImporting}
          >
            {isImporting ? 'Importing...' : 'Import'}
          </button>
        </div>
      </div>
    </div>
  );
};