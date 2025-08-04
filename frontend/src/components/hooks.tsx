/**
 * Copyright 2025 Clidey, Inc.
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

import { useCallback, useEffect, useRef, useState } from "react";


export const useExportToCSV = (schema: string, storageUnit: string, selectedOnly: boolean = false, delimiter: string = ',', selectedRows?: Record<string, any>[]) => {
    return useCallback(async () => {
      try {
        // Prepare request body
        const requestBody: any = {
          schema,
          storageUnit,
          delimiter,
        };
        
        // Add selected rows if provided
        // For now, we'll use the full row data approach for selections
        // In the future, we can optimize this by detecting primary keys
        if (selectedOnly && selectedRows && selectedRows.length > 0) {
          // Threshold: if more than 1000 rows selected, warn the user
          if (selectedRows.length > 1000) {
            console.warn(`Exporting ${selectedRows.length} rows. Large selections may be slow.`);
          }
          requestBody.selectedRows = selectedRows;
        }
        
        // Use backend export endpoint for full data export
        const response = await fetch('/api/export', {
          method: 'POST',
          credentials: 'include',
          headers: {
            'Accept': 'text/csv',
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
          throw new Error(`Export failed: ${response.statusText}`);
        }

        // Get filename from Content-Disposition header
        const contentDisposition = response.headers.get('Content-Disposition');
        let filename = `${schema}_${storageUnit}.csv`;
        if (contentDisposition) {
          const filenameMatch = contentDisposition.match(/filename="(.+)"/);
          if (filenameMatch) {
            filename = filenameMatch[1];
          }
        }

        // Create blob from response
        const blob = await response.blob();
        
        // Create download link
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
      } catch (error) {
        console.error('Export failed:', error);
        throw error;
      }
    }, [schema, storageUnit, selectedOnly, delimiter, selectedRows]);
};

type ILongPressProps = {
  onLongPress: () => (() => void | void);
  onClick?: () => void;
  ms?: number;
}
  
export const useLongPress = ({
  onLongPress,
  onClick = () => {},
  ms = 300,
}: ILongPressProps) => {
  const [startLongPress, setStartLongPress] = useState(false);
  const cleanUpFunction = useRef<() => void | void>();

  useEffect(() => {
    let timerId: ReturnType<typeof setTimeout> | undefined;
    if (startLongPress) {
      timerId = setTimeout(() => {
        cleanUpFunction.current = onLongPress();
        setStartLongPress(false);
      }, ms);
    } else {
      if (timerId != null) {
        clearTimeout(timerId);
      }
    }

    return () => {
      if (timerId != null) {
        clearTimeout(timerId);
      }
    };
  }, [onLongPress, ms, startLongPress]);

  const start = useCallback(() => {
    setStartLongPress(true);
  }, []);

  const stop = useCallback(() => {
    if (startLongPress) {
      onClick?.();
    }
    cleanUpFunction.current?.();
    setStartLongPress(false);
  }, [onClick, startLongPress]);

  return {
    onMouseDown: start,
    onMouseUp: stop,
    onTouchStart: start,
    onTouchEnd: stop,
  };
};