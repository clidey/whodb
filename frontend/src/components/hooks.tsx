import { MouseEvent, TouchEvent, useCallback, useRef, useState } from "react";


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
  if (!isTouchEvent(e)) return;
  if (e.touches.length < 2 && e.preventDefault != null) {
    e.preventDefault();
  }
}

function isTouchEvent<T extends unknown = any>(e: MouseEvent<T> | TouchEvent<T> | Event): e is TouchEvent<T> {
  return e && "touches" in e;
}

type IPressHandlers<T> = {
  onLongPress: (e: MouseEvent<T> | TouchEvent<T>) => void | (() => void),
  onClick?: (e: MouseEvent<T> | TouchEvent<T>) => void,
}

type IOptions = {
  delay?: number;
  shouldPreventDefault?: boolean;
}

export const useLongPress = <T extends unknown = any>(
  { onLongPress, onClick }: IPressHandlers<T>,
  { delay = 300, shouldPreventDefault = true }: IOptions = {}
) => {
  const [longPressTriggered, setLongPressTriggered] = useState(false);
  const timeout = useRef<NodeJS.Timeout>();
  const target = useRef<EventTarget>();
  const cleanUpFunction = useRef<Function>();

  const start = useCallback(
    (e: MouseEvent<T> | TouchEvent<T>) => {
      e.persist();
      const clonedEvent = { ...e };
      
      if (shouldPreventDefault && e.target) {
        e.target.addEventListener("touchend", preventDefault, { passive: false });
        target.current = e.target;
      }

      timeout.current = setTimeout(() => {
        const cleanup = onLongPress(clonedEvent);
        if (cleanup != null) {
          cleanUpFunction.current = cleanup;
        }
        setLongPressTriggered(true);
      }, delay);

      if (!isTouchEvent(e)) {
        document.addEventListener("mousemove", e => e.preventDefault());
      }
    },
    [onLongPress, delay, shouldPreventDefault]
  );

  const clear = useCallback(
    (e: MouseEvent<T> | TouchEvent<T>, shouldTriggerClick = true) => {
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

      if (!isTouchEvent(e)) {
        document.removeEventListener("mousemove", e => e.preventDefault());
      }
    },
    [shouldPreventDefault, onClick, longPressTriggered]
  );

  return {
    onMouseDown: (e: MouseEvent<T>) => start(e),
    onTouchStart: (e: TouchEvent<T>) => start(e),
    onMouseUp: (e: MouseEvent<T>) => clear(e),
    onMouseLeave: (e: MouseEvent<T>) => clear(e, false),
    onTouchEnd: (e: TouchEvent<T>) => clear(e)
  };
};