/*
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

import {useCallback, useEffect, useRef, useState} from "react";
import * as desktopService from "../services/desktop";
import { isDesktopApp } from "../utils/external-links";
import { addAuthHeader } from "../utils/auth-headers";


export const useExportToCSV = (schema: string, storageUnit: string, selectedOnly: boolean = false, delimiter: string = ',', selectedRows?: Record<string, any>[], format: 'csv' | 'excel' = 'csv') => {
    return useCallback(async () => {
      try {
        // Prepare request body
        const requestBody: any = {
          schema,
          storageUnit,
          delimiter,
          format,
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
        // Add auth header for desktop environments where cookies don't work
        const response = await fetch('/api/export', {
          method: 'POST',
          credentials: 'include',
          headers: addAuthHeader({
            'Accept': format === 'excel' ? 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' : 'text/csv',
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
          throw new Error(await response.text());
        }

        // Get filename from Content-Disposition header
        const contentDisposition = response.headers.get('Content-Disposition');
        // Only include schema in filename if it exists (for SQLite, schema is empty)
        let filename = schema ? `${schema}_${storageUnit}.${format === 'excel' ? 'xlsx' : 'csv'}` : `${storageUnit}.${format === 'excel' ? 'xlsx' : 'csv'}`;
        if (contentDisposition) {
          const filenameMatch = contentDisposition.match(/filename="(.+)"/);
          if (filenameMatch) {
            filename = filenameMatch[1];
          }
        }

        // Create blob from response
        const blob = await response.blob();

        // Use native save dialog in desktop mode
        if (isDesktopApp()) {
          const arrayBuffer = await blob.arrayBuffer();
          const data = new Uint8Array(arrayBuffer);
          console.log('Export: Attempting to save file', {
            filename,
            dataSize: data.length,
            format,
            isDesktop: isDesktopApp()
          });
          const savedPath = await desktopService.saveBinaryFile(data, filename);
          if (!savedPath) {
            console.error('Export: Save dialog was cancelled or failed');
            // Don't throw error if user just cancelled the dialog
            return;
          }
          console.log('Export: File saved successfully to', savedPath);
        } else {
          // Browser download fallback
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
        }
      } catch (error) {
        throw error;
      }
    }, [schema, storageUnit, selectedOnly, delimiter, selectedRows, format]);
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