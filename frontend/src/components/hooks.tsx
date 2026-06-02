/*
 * Copyright 2026 Clidey, Inc.
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

import type { SourceObjectRefInput } from "@graphql";
import {useCallback} from "react";
import * as desktopService from "../services/desktop";
import {isDesktopApp} from "../utils/external-links";
import {addAuthHeader} from "../utils/auth-headers";
import {withBasePath} from "../utils/base-path";


/**
 * Exports the current source object selection or full object contents through the backend export endpoint.
 */
export const useExportToCSV = (
    objectRef: SourceObjectRefInput | undefined,
    fileBaseName: string,
    selectedOnly: boolean = false,
    delimiter: string = ',',
    selectedRows?: Record<string, any>[],
    format: 'csv' | 'excel' | 'ndjson' = 'csv'
) => {
    return useCallback(async () => {
        // Prepare request body
        const requestBody: any = {
          ref: objectRef,
          fileBaseName,
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
        const response = await fetch(withBasePath('/api/export'), {
          method: 'POST',
          credentials: 'include',
          headers: addAuthHeader({
            'Accept': format === 'excel' ? 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' : format === 'ndjson' ? 'application/x-ndjson' : 'text/csv',
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
          throw new Error(await response.text());
        }

        // Get filename from Content-Disposition header
        const contentDisposition = response.headers.get('Content-Disposition');
        const extension = format === 'excel' ? 'xlsx' : format === 'ndjson' ? 'ndjson' : 'csv';
        let filename = `${fileBaseName}.${extension}`;
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
          const savedPath = await desktopService.saveBinaryFile(data, filename);
          if (!savedPath) {
            console.error('Export: Save dialog was cancelled or failed');
            // Don't throw error if user just cancelled the dialog
            return;
          }
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
    }, [objectRef, fileBaseName, selectedOnly, delimiter, selectedRows, format]);
};
