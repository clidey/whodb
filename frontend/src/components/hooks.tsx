import { useCallback, useEffect, useRef, useState } from "react";


export const useExportToCSV = (columns: string[], rows: Record<string, string>[]) => {
    return useCallback(() => {
      const csvContent = [
        columns.join(','), 
        ...rows.map(row => columns.map(col => row[col]).join(","))
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
    }, [columns, rows]);
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