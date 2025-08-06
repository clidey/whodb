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

import { FC, useState, useRef, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import classNames from "classnames";
import { AnimatedButton } from "./button";
import { Icons } from "./icons";
import { InputWithlabel } from "./input";
import { Dropdown } from "./dropdown";
import { StorageUnit, useGenerateMockDataMutation } from "@graphql";
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
    const [progress, setProgress] = useState<{
        total: number;
        generated: number;
        failed: number;
    } | null>(null);

    // AI providers - currently disabled as per requirements
    const aiProviders = [
        { label: "Normal", value: "Normal", disabled: false },
        { label: "ChatGPT", value: "ChatGPT", disabled: true },
        { label: "Claude", value: "Claude", disabled: true },
        { label: "Ollama", value: "Ollama", disabled: true },
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

    const handleGenerate = async () => {
        if (overwriteExisting === "overwrite" && !showConfirmation) {
            setShowConfirmation(true);
            return;
        }

        const count = parseInt(rowCount) || 100;
        
        try {
            setProgress({ total: count, generated: 0, failed: 0 });
            
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
            if (data) {
                setProgress({
                    total: data.TotalRows,
                    generated: data.GeneratedRows,
                    failed: data.FailedRows,
                });

                if (data.FailedRows > 0 && data.ErrorMessages.length > 0) {
                    notify(`Generated ${data.GeneratedRows} rows successfully. ${data.FailedRows} rows failed. Errors: ${data.ErrorMessages.join(", ")}`, "warning");
                } else {
                    notify(`Successfully generated ${data.GeneratedRows} rows`, "success");
                }

                onSuccess();
                onClose();
            }
        } catch (error: any) {
            if (error.message === "mock data generation is not allowed for this table") {
                notify("Mock data generation is not allowed for this table", "error");
            } else {
                notify(`Failed to generate mock data: ${error.message}`, "error");
            }
            setProgress(null);
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

                        {!showConfirmation ? (
                            <div className="space-y-4">
                                <InputWithlabel
                                    label="Number of Rows"
                                    value={rowCount}
                                    setValue={setRowCount}
                                    type="number"
                                    placeholder="Enter number of rows"
                                />

                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        Method
                                    </label>
                                    <Dropdown
                                        options={aiProviders}
                                        selected={method}
                                        setSelected={setMethod}
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        Data Handling
                                    </label>
                                    <Dropdown
                                        options={[
                                            { label: "Append to existing data", value: "append" },
                                            { label: "Overwrite existing data", value: "overwrite" },
                                        ]}
                                        selected={overwriteExisting}
                                        setSelected={setOverwriteExisting}
                                    />
                                </div>

                                {loading && progress && (
                                    <div className="mt-4">
                                        <div className="flex justify-between text-sm text-gray-600 dark:text-gray-400 mb-1">
                                            <span>Generating data...</span>
                                            <span>{progress.generated} / {progress.total}</span>
                                        </div>
                                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                                            <div
                                                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                                                style={{ width: `${(progress.generated / progress.total) * 100}%` }}
                                            />
                                        </div>
                                        {progress.failed > 0 && (
                                            <p className="text-sm text-red-500 mt-1">
                                                {progress.failed} rows failed
                                            </p>
                                        )}
                                    </div>
                                )}

                                <div className="flex justify-end gap-2 mt-6">
                                    <AnimatedButton
                                        type="md"
                                        label="Cancel"
                                        onClick={handleCancel}
                                        disabled={loading}
                                        className="bg-gray-200 dark:bg-gray-700"
                                    />
                                    <AnimatedButton
                                        type="md"
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
                                        type="md"
                                        label="Cancel"
                                        onClick={() => setShowConfirmation(false)}
                                        disabled={loading}
                                        className="bg-gray-200 dark:bg-gray-700"
                                    />
                                    <AnimatedButton
                                        type="md"
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