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


export const useExportToCSV = (columns: string[], rows: Record<string, string | number>[], specificIndexes: number[] = []) => {
    return useCallback(() => {
      let selectedRows: Record<string, string | number>[];
      if (specificIndexes.length === 0) {
        selectedRows = rows;
      } else {
        selectedRows = specificIndexes.map(index => rows[index]);
      }
      const csvContent = [
        columns.join(','), 
        ...selectedRows.map(row => columns.map(col => row[col]).join(","))
      ].join('\n'); 
  
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  
      const link = document.createElement('a');
      if (link.download !== undefined) {
        const url = URL.createObjectURL(blob);
        link.setAttribute('href', url);
        link.setAttribute('download', 'data.csv');
        link.style.visibility = 'hidden';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
      }
    }, [columns, rows, specificIndexes]);
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
    let timerId: NodeJS.Timeout | undefined;
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