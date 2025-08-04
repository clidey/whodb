import { useState, useCallback } from 'react';
import { notify } from '../store/function';

interface ImportResult {
  processedRows: number;
  status: 'completed' | 'failed';
  error?: string;
}

export const useCSVImport = (
  schema: string,
  storageUnit: string,
  onComplete?: () => void
) => {
  const [isImporting, setIsImporting] = useState(false);
  const [progress, setProgress] = useState(0);

  const importCSV = useCallback(async (
    file: File,
    mode: 'append' | 'override' = 'append'
  ): Promise<ImportResult | null> => {
    // Validate file size on client
    const MAX_SIZE = 50 * 1024 * 1024; // 50MB
    if (file.size > MAX_SIZE) {
      notify(`File too large. Maximum size is ${MAX_SIZE / (1024 * 1024)}MB`, 'error');
      return null;
    }

    setIsImporting(true);
    setProgress(0);

    const formData = new FormData();
    formData.append('file', file);
    formData.append('schema', schema);
    formData.append('storageUnit', storageUnit);
    formData.append('mode', mode);

    try {
      const response = await fetch('/api/import', {
        method: 'POST',
        body: formData,
        credentials: 'include', // Include auth cookies
      });

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.error || `Import failed: ${response.statusText}`);
      }

      if (result.status === 'completed') {
        notify(`Successfully imported ${result.processedRows} rows`, 'success');
        onComplete?.();
      }

      return result;
    } catch (error: any) {
      notify(`Import failed: ${error.message}`, 'error');
      return {
        processedRows: 0,
        status: 'failed',
        error: error.message
      };
    } finally {
      setIsImporting(false);
      setProgress(0);
    }
  }, [schema, storageUnit, onComplete]);

  return {
    importCSV,
    isImporting,
    progress
  };
};