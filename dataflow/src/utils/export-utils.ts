import * as XLSX from 'xlsx';

/**
 * Triggers a browser file download from a Blob.
 */
export function downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

/**
 * Converts query result columns + rows to a CSV Blob (RFC 4180).
 */
export function toCSV(
    columns: Array<{ Name: string }>,
    rows: string[][]
): Blob {
    const escapeField = (field: string): string => {
        if (field.includes(',') || field.includes('"') || field.includes('\n') || field.includes('\r')) {
            return '"' + field.replace(/"/g, '""') + '"';
        }
        return field;
    };

    const header = columns.map(c => escapeField(c.Name)).join(',');
    const dataLines = rows.map(row => row.map(escapeField).join(','));
    const csv = [header, ...dataLines].join('\r\n');
    return new Blob([csv], { type: 'text/csv;charset=utf-8' });
}

/**
 * Converts query result columns + rows to a JSON Blob.
 * Output: array of objects [{col1: val1, col2: val2}, ...]
 */
export function toJSON(
    columns: Array<{ Name: string }>,
    rows: string[][]
): Blob {
    const objects = rows.map(row => {
        const obj: Record<string, string> = {};
        columns.forEach((col, i) => {
            obj[col.Name] = row[i];
        });
        return obj;
    });
    const json = JSON.stringify(objects, null, 2);
    return new Blob([json], { type: 'application/json;charset=utf-8' });
}

/**
 * Converts query result columns + rows to an Excel (.xlsx) Blob.
 */
export function toExcel(
    sheetName: string,
    columns: Array<{ Name: string }>,
    rows: string[][]
): Blob {
    const header = columns.map(c => c.Name);
    const data = [header, ...rows];
    const worksheet = XLSX.utils.aoa_to_sheet(data);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, sheetName.slice(0, 31));
    const buffer = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' });
    return new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
}

/**
 * Converts an SVG data URL to a PNG Blob via offscreen canvas.
 */
export function svgDataURLToPNG(
    svgDataURL: string,
    width: number,
    height: number,
    pixelRatio: number = 2
): Promise<Blob> {
    return new Promise((resolve, reject) => {
        const img = new Image();
        img.onload = () => {
            const canvas = document.createElement('canvas');
            canvas.width = width * pixelRatio;
            canvas.height = height * pixelRatio;
            const ctx = canvas.getContext('2d');
            if (!ctx) {
                reject(new Error('Failed to get canvas 2d context'));
                return;
            }
            ctx.scale(pixelRatio, pixelRatio);
            ctx.fillStyle = '#ffffff';
            ctx.fillRect(0, 0, width, height);
            ctx.drawImage(img, 0, 0, width, height);
            canvas.toBlob(blob => {
                if (blob) resolve(blob);
                else reject(new Error('Failed to convert canvas to PNG blob'));
            }, 'image/png');
        };
        img.onerror = () => reject(new Error('Failed to load SVG image'));
        img.src = svgDataURL;
    });
}
