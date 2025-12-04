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

import {
    Button,
    Label,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Sheet,
    SheetContent,
    SheetFooter,
    SheetTitle,
    toast
} from "@clidey/ux";
import {FC, useCallback, useMemo, useState} from "react";
import {useExportToCSV} from "./hooks";
import { ShareIcon } from "./heroicons";
import { VisuallyHidden } from "@radix-ui/react-visually-hidden";
import { useTranslation } from "@/hooks/use-translation";

interface IExportProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    schema: string;
    storageUnit: string;
    hasSelectedRows: boolean;
    selectedRowsData?: Record<string, any>[];
    checkedRowsCount: number;
}

export const Export: FC<IExportProps> = ({
                                             open,
                                             onOpenChange,
                                             schema,
                                             storageUnit,
                                             hasSelectedRows,
                                             selectedRowsData,
                                             checkedRowsCount,
                                         }) => {
    const { t } = useTranslation('components/export');
    const [exportDelimiter, setExportDelimiter] = useState(',');
    const [exportFormat, setExportFormat] = useState<'csv' | 'excel'>('csv');

    // Export options as lists - CE version only has basic download
    const exportFormatOptions = useMemo(() => [
        {value: 'csv', label: t('formatCsv')},
        {value: 'excel', label: t('formatExcel')},
    ] as const, [t]);

    const exportDelimiterOptions = useMemo(() => [
        {value: ',', label: t('delimiterComma')},
        {value: ';', label: t('delimiterSemicolon')},
        {value: '|', label: t('delimiterPipe')},
        {value: '\t', label: t('delimiterTab')},
    ] as const, [t]);

    // Selected rows are already in the correct format for the hook.
    const selectedRowsForExport = useMemo(() => {
        if (!hasSelectedRows || !selectedRowsData) {
            return undefined;
        }
        return selectedRowsData;
    }, [hasSelectedRows, selectedRowsData]);

    // Always call the hook, but use conditional logic inside
    const backendExport = useExportToCSV(
        schema || '',
        storageUnit || '',
        hasSelectedRows,
        exportDelimiter,
        selectedRowsForExport,
        exportFormat
    );

    const handleExportConfirm = useCallback(async () => {
        try {
            await backendExport();
            onOpenChange(false);
        } catch (error: any) {
            toast.error(error.message || t('exportFailed'));
        }
    }, [backendExport, onOpenChange, t]);

    return (
        <>
            <Sheet open={open} onOpenChange={onOpenChange}>
                <SheetContent side="right" className="max-w-md w-full p-8">
                    <SheetTitle className="flex items-center gap-2"><ShareIcon className="w-4 h-4" /> {t('title')}</SheetTitle>
                    <VisuallyHidden>
                        <SheetTitle>{t('title')}</SheetTitle>
                    </VisuallyHidden>
                    <div className="flex flex-col gap-lg grow">
                        <div className="space-y-4 grow">
                            <p>
                                {hasSelectedRows
                                    ? t('selectedRows', { count: checkedRowsCount })
                                    : t('allData')}
                            </p>
                            <div className="mb-4 flex flex-col gap-2">
                                <Label>
                                    {t('format')}
                                </Label>
                                <Select
                                    value={exportFormat}
                                    onValueChange={(value) => setExportFormat(value as 'csv' | 'excel')}
                                >
                                    <SelectTrigger className="w-full">
                                        <SelectValue>
                                            {
                                                exportFormatOptions.find(opt => opt.value === exportFormat)?.label
                                            }
                                        </SelectValue>
                                    </SelectTrigger>
                                    <SelectContent>
                                        {exportFormatOptions.map(opt => (
                                            <SelectItem key={opt.value} value={opt.value} data-value={opt.value}>
                                                {opt.label}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                                {exportFormat === 'csv' && (
                                    <>
                                        <Label>
                                            {t('delimiter')}
                                        </Label>
                                        <Select
                                            value={exportDelimiter}
                                            onValueChange={(value) => setExportDelimiter(value)}
                                        >
                                            <SelectTrigger className="w-full">
                                                <SelectValue>
                                                    {
                                                        exportDelimiterOptions.find(opt => opt.value === exportDelimiter)?.label
                                                    }
                                                </SelectValue>
                                            </SelectTrigger>
                                            <SelectContent>
                                                {exportDelimiterOptions.map(opt => (
                                                    <SelectItem key={opt.value} value={opt.value} data-value={opt.value}>
                                                        {opt.label}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        <p className="text-sm mt-2">{t('delimiterHelp')}</p>
                                    </>
                                )}
                            </div>
                        </div>
                        <SheetFooter className="flex gap-sm px-0">
                            <div className="text-xs text-muted-foreground mb-8">
                                <p className="font-medium mb-1">{t('exportDetailsTitle')}</p>
                                <ul className="list-disc list-inside space-y-1">
                                    {exportFormat === 'csv' ? (
                                        <>
                                            <li><p className="inline-block">{t('csvHeaders')}</p></li>
                                            <li><p className="inline-block">{t('csvEncoding')}</p></li>
                                            <li><p className="inline-block">{t('csvDelimiter')}</p></li>
                                        </>
                                    ) : (
                                        <>
                                            <li><p className="inline-block">{t('excelFormat')}</p></li>
                                            <li><p className="inline-block">{t('excelHeaders')}</p></li>
                                            <li><p className="inline-block">{t('excelColumns')}</p></li>
                                        </>
                                    )}
                                </ul>
                            </div>
                            <div className="flex flex-row gap-sm">
                                <Button
                                    className="flex-1"
                                    variant="secondary"
                                    onClick={() => onOpenChange(false)}
                                    data-testid="cancel-export"
                                >
                                    {t('cancel')}
                                </Button>
                                <Button className="flex-1" onClick={handleExportConfirm}>
                                    {t('export')}
                                </Button>
                            </div>
                        </SheetFooter>
                    </div>
                </SheetContent>
            </Sheet>
        </>
    );
};