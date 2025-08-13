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

import { FC, useState, useRef, useEffect, cloneElement } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { AnimatedButton } from "./button";
import { Icons } from "./icons";
import { InputWithlabel, ToggleInput } from "./input";
import { Dropdown, createDropdownItem, IDropdownItem } from "./dropdown";
import { StorageUnit, useGenerateMockDataMutation, useMockDataMaxRowCountQuery } from "@graphql";
import { notify } from "../store/function";
import { useAppSelector } from "../store/hooks";

interface MockDataDialogProps {
    isOpen: boolean;
    onClose: () => void;
    storageUnit: StorageUnit;
    onSuccess: () => void;
}

export const MockDataDialog: FC<MockDataDialogProps> = ({ isOpen, onClose, storageUnit, onSuccess }) => {
    const [rowCount, setRowCount] = useState("100");
    const [method, setMethod] = useState("Normal");
    const [overwriteExisting, setOverwriteExisting] = useState("append");
    const [showConfirmation, setShowConfirmation] = useState(false);
    const dialogRef = useRef<HTMLDivElement>(null);
    const schema = useAppSelector(state => state.database.schema);
    const [generateMockData, { loading }] = useGenerateMockDataMutation();
    const { data: maxRowData } = useMockDataMaxRowCountQuery();
    const maxRowCount = maxRowData?.MockDataMaxRowCount || 200;

    const methodItems: IDropdownItem[] = [createDropdownItem("Normal")];
    const handlingItems: IDropdownItem[] = [
        { id: "append", label: "Append to existing data" },
        { id: "overwrite", label: "Overwrite existing data" },
    ];

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dialogRef.current && !dialogRef.current.contains(event.target as Node)) {
                if (!loading) {
                    onClose();
                }
            }
        };

        const handleEscape = (event: KeyboardEvent) => {
            if (event.key === 'Escape' && !loading) {
                onClose();
            }
        };

        if (isOpen) {
            document.addEventListener('mousedown', handleClickOutside);
            document.addEventListener('keydown', handleEscape);
        }

        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
            document.removeEventListener('keydown', handleEscape);
        };
    }, [isOpen, onClose, loading]);

    const handleRowCountChange = (value: string) => {
        // Only allow numeric input
        const numericValue = value.replace(/[^0-9]/g, '');
        const parsedValue = parseInt(numericValue) || 0;
        
        // Enforce max limit
        if (parsedValue > maxRowCount) {
            setRowCount(maxRowCount.toString());
            notify(`Maximum row count is ${maxRowCount}`, "warning");
        } else {
            setRowCount(numericValue);
        }
    };

    const handleGenerate = async () => {
        if (overwriteExisting === "overwrite" && !showConfirmation) {
            setShowConfirmation(true);
            return;
        }

        const count = parseInt(rowCount) || 100;
        
        // Double-check the limit
        if (count > maxRowCount) {
            notify(`Row count cannot exceed ${maxRowCount}`, "error");
            return;
        }
        
        try {
            const result = await generateMockData({
                variables: {
                    input: {
                        Schema: schema,
                        StorageUnit: storageUnit.Name,
                        RowCount: count,
                        Method: method,
                        OverwriteExisting: overwriteExisting === "overwrite",
                    }
                }
            });

            const data = result.data?.GenerateMockData;
            if (data?.Status) {
                notify(`Successfully generated ${count} rows`, "success");
                onSuccess();
                onClose();
            } else {
                notify(`Failed to generate mock data`, "error");
            }
        } catch (error: any) {
            if (error.message === "mock data generation is not allowed for this table") {
                notify("Mock data generation is not allowed for this table", "error");
            } else {
                notify(`Failed to generate mock data: ${error.message}`, "error");
            }
        }
    };

    const handleCancel = () => {
        if (!loading) {
            setShowConfirmation(false);
            onClose();
        }
    };

    return (
        <AnimatePresence>
            {isOpen && (
                <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
                >
                    <motion.div
                        ref={dialogRef}
                        initial={{ scale: 0.9, opacity: 0 }}
                        animate={{ scale: 1, opacity: 1 }}
                        exit={{ scale: 0.9, opacity: 0 }}
                        className="bg-white dark:bg-gray-800 rounded-lg p-6 w-96 max-w-[90vw] shadow-xl"
                    >
                        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
                            Generate Mock Data for {storageUnit.Name}
                        </h2>

                        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md p-3 mb-4">
                            <p className="text-sm text-blue-700 dark:text-blue-300">
                                <strong>Note:</strong> Mock data generation is in development. You may experience some errors or missing data.
                            </p>
                        </div>

                        {!showConfirmation ? (
                            <div className="space-y-4">
                                <div>
                                    <InputWithlabel
                                        label={`Number of Rows (max: ${maxRowCount})`}
                                        value={rowCount}
                                        setValue={handleRowCountChange}
                                        type="text"
                                        inputProps={{ 
                                            inputMode: "numeric", 
                                            pattern: "[0-9]*",
                                            max: maxRowCount.toString()
                                        }}
                                        placeholder={`Enter number of rows (1-${maxRowCount})`}
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        Method
                                    </label>
                                    <Dropdown
                                        items={methodItems}
                                        value={methodItems.find(i => i.id === method)}
                                        onChange={(item) => setMethod(item.id)}
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        Data Handling
                                    </label>
                                    <Dropdown
                                        items={handlingItems}
                                        value={handlingItems.find(i => i.id === overwriteExisting)}
                                        onChange={(item) => setOverwriteExisting(item.id)}
                                    />
                                </div>

                                {loading && (
                                    <div className="mt-4">
                                        <div className="flex justify-center">
                                            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                                        </div>
                                        <p className="text-center text-sm text-gray-600 dark:text-gray-400 mt-2">
                                            Generating mock data...
                                        </p>
                                    </div>
                                )}

                                <div className="flex justify-end gap-2 mt-6">
                                    <AnimatedButton
                                        type="sm"
                                        label="Cancel"
                                        onClick={handleCancel}
                                        disabled={loading}
                                        className="bg-gray-200 dark:bg-gray-700"
                                        icon={Icons.Cancel}
                                    />
                                    <AnimatedButton
                                        type="sm"
                                        icon={Icons.CheckCircle}
                                        label="Generate"
                                        onClick={handleGenerate}
                                        disabled={loading}
                                        className="bg-blue-600 text-white"
                                    />
                                </div>
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <div className="flex items-center justify-center mb-4">
                                    <div className="w-16 h-16 bg-yellow-100 dark:bg-yellow-900 rounded-full flex items-center justify-center">
                                        {cloneElement(Icons.Warning, { className: "w-8 h-8 text-yellow-600 dark:text-yellow-400" })}
                                    </div>
                                </div>
                                <p className="text-center text-gray-700 dark:text-gray-300">
                                    Are you sure you want to overwrite all existing data in {storageUnit.Name}? This action cannot be undone.
                                </p>
                                <div className="flex justify-end gap-2 mt-6">
                                    <AnimatedButton
                                        type="sm"
                                        label="Cancel"
                                        onClick={() => setShowConfirmation(false)}
                                        disabled={loading}
                                        className="bg-gray-200 dark:bg-gray-700"
                                        icon={Icons.Cancel}
                                    />
                                    <AnimatedButton
                                        type="sm"
                                        icon={Icons.Delete}
                                        label="Yes, Overwrite"
                                        onClick={handleGenerate}
                                        disabled={loading}
                                        className="bg-red-600 text-white"
                                    />
                                </div>
                            </div>
                        )}
                    </motion.div>
                </motion.div>
            )}
        </AnimatePresence>
    );
};