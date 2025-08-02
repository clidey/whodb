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


export const useExportToCSV = (schema: string, storageUnit: string, selectedOnly: boolean = false) => {
    return useCallback(() => {
      // Use backend export endpoint for full data export
      const params = new URLSearchParams({
        schema,
        storageUnit,
      });
      
      const url = `/api/export-csv?${params}`;
      
      // Create link and trigger download
      const link = document.createElement('a');
      link.href = url;
      link.style.visibility = 'hidden';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }, [schema, storageUnit, selectedOnly]);
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