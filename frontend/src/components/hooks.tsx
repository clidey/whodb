import { useCallback, useRef, useState } from "react";


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
  
  
function preventDefault(e: Event) {
  if ( !isTouchEvent(e) ) return;
  if (e.touches.length < 2 && e.preventDefault) {
    e.preventDefault();
  }
};

function isTouchEvent(e: Event): e is TouchEvent {
  return e && "touches" in e;
};

type IPressHandlers<T> = {
  onLongPress: (e: React.MouseEvent<T> | React.TouchEvent<T>) => void | (() => void),
  onClick?: (e: React.MouseEvent<T> | React.TouchEvent<T>) => void,
}

type IOptions = {
  delay?: number;
  shouldPreventDefault?: boolean;
}

export const useLongPress = <T extends unknown = any,>(
  { onLongPress, onClick }: IPressHandlers<T>,
  { delay = 300, shouldPreventDefault = true }
  : IOptions
  = {}
) => {
  const [longPressTriggered, setLongPressTriggered] = useState(false);
  const timeout = useRef<NodeJS.Timeout>();
  const target = useRef<EventTarget>();
  const cleanUpFunction = useRef<Function>();

  const start = useCallback(
    (e: React.MouseEvent<T> | React.TouchEvent<T>) => {
      e.persist();
      const clonedEvent = {...e};
      
      if (shouldPreventDefault && e.target) {
        e.target.addEventListener(
          "touchend",
          preventDefault,
          { passive: false }
        );
        target.current = e.target;
      }

      timeout.current = setTimeout(() => {
        const cleanup = onLongPress(clonedEvent);
        if (cleanup != null) {
          cleanUpFunction.current = cleanup;
        }
        setLongPressTriggered(true);
      }, delay);
    },
    [onLongPress, delay, shouldPreventDefault]
  );

  const clear = useCallback((
      e: React.MouseEvent<T> | React.TouchEvent<T>,
      shouldTriggerClick = true
    ) => {
      if (timeout.current != null) {
        clearTimeout(timeout.current);
      }

      if (cleanUpFunction.current != null) {
        cleanUpFunction.current();
        cleanUpFunction.current = undefined;
      }
      
      if (shouldTriggerClick && !longPressTriggered) {
        onClick?.(e);
      }

      setLongPressTriggered(false);

      if (shouldPreventDefault && target.current) {
        target.current.removeEventListener("touchend", preventDefault);
      }
    },
    [shouldPreventDefault, onClick, longPressTriggered]
  );

  return {
    onMouseDown: (e: React.MouseEvent<T>) => start(e),
    onTouchStart: (e: React.TouchEvent<T>) => start(e),
    onMouseUp: (e: React.MouseEvent<T>) => clear(e),
    onMouseLeave: (e: React.MouseEvent<T>) => clear(e, false),
    onTouchEnd: (e: React.TouchEvent<T>) => clear(e)
  };
};