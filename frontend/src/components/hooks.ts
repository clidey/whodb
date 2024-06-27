import { useCallback } from "react";

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
  