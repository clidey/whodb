/*
 * Copyright 2026 Clidey, Inc.
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

import { RefObject, useEffect, useState } from "react";

/**
 * Tracks the offsetWidth of a container element, updating on window resize.
 * Returns the current width as a number (0 until the element is mounted).
 *
 * The returned value can be used as a React `key` on child components to force
 * a full remount when the container width changes (e.g. virtualised tables that
 * capture their width once on mount).
 */
export function useContainerWidth(ref: RefObject<HTMLElement | null>): number {
    const [width, setWidth] = useState(0);

    useEffect(() => {
        const update = () => {
            if (ref.current) {
                setWidth(ref.current.offsetWidth);
            }
        };

        update();
        window.addEventListener('resize', update);
        return () => window.removeEventListener('resize', update);
    }, [ref]);

    return width;
}
