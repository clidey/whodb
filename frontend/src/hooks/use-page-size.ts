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

import {useCallback, useState} from "react";
import {toast} from "@clidey/ux";

export const PRESET_PAGE_SIZES = ["10", "25", "50", "100", "250", "500", "1000"];

interface UsePageSizeOptions {
    onPageSizeChange?: (size: number) => void;
}

export function usePageSize(initialSize: number, options?: UsePageSizeOptions) {
    const [pageSize, setPageSize] = useState(String(initialSize));
    const [isCustom, setIsCustom] = useState(!PRESET_PAGE_SIZES.includes(String(initialSize)));
    const [customInput, setCustomInput] = useState(String(initialSize));

    const handleSelectChange = useCallback((value: string) => {
        if (value === "custom") {
            setIsCustom(true);
            setCustomInput(pageSize);
        } else {
            setIsCustom(false);
            setPageSize(value);
            options?.onPageSizeChange?.(Number.parseInt(value, 10));
        }
    }, [pageSize, options]);

    const handleCustomApply = useCallback(() => {
        const parsed = Number.parseInt(customInput, 10);
        if (Number.isNaN(parsed) || parsed <= 0) {
            toast.error("Please enter a number greater than 0");
            setCustomInput(pageSize);
            return false;
        }
        if (parsed > 20000) {
            toast.warning("Large page sizes may result in slower performance");
        }
        setPageSize(String(parsed));
        options?.onPageSizeChange?.(parsed);
        return true;
    }, [customInput, pageSize, options]);

    return {
        pageSize: Number.parseInt(pageSize, 10),
        pageSizeString: pageSize,
        isCustom,
        customInput,
        setCustomInput,
        handleSelectChange,
        handleCustomApply,
    };
}
