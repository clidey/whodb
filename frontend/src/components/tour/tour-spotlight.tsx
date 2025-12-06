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

import { motion, AnimatePresence } from 'framer-motion';
import { FC, useEffect, useState } from 'react';

interface TourSpotlightProps {
    targetElement: HTMLElement | null;
    padding?: number;
}

export const TourSpotlight: FC<TourSpotlightProps> = ({ targetElement, padding = 8 }) => {
    const [rect, setRect] = useState<DOMRect | null>(null);

    useEffect(() => {
        if (!targetElement) {
            setRect(null);
            return;
        }

        const updateRect = () => {
            setRect(targetElement.getBoundingClientRect());
        };

        updateRect();
        window.addEventListener('resize', updateRect);
        window.addEventListener('scroll', updateRect, true);

        return () => {
            window.removeEventListener('resize', updateRect);
            window.removeEventListener('scroll', updateRect, true);
        };
    }, [targetElement]);

    if (!rect) return null;

    const highlightRect = {
        left: rect.left - padding,
        top: rect.top - padding,
        width: rect.width + padding * 2,
        height: rect.height + padding * 2,
    };

    return (
        <>
            <motion.svg
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.3, ease: "easeInOut" }}
                className="fixed inset-0 z-[9998] pointer-events-none"
                style={{ width: '100vw', height: '100vh' }}
            >
                <defs>
                    <mask id={`tour-spotlight-mask-${rect.left}-${rect.top}`}>
                        <rect x="0" y="0" width="100%" height="100%" fill="white" />
                        <rect
                            x={highlightRect.left}
                            y={highlightRect.top}
                            width={highlightRect.width}
                            height={highlightRect.height}
                            rx="8"
                            fill="black"
                        />
                    </mask>
                </defs>
                <rect
                    x="0"
                    y="0"
                    width="100%"
                    height="100%"
                    fill="rgba(0, 0, 0, 0.7)"
                    mask={`url(#tour-spotlight-mask-${rect.left}-${rect.top})`}
                />
            </motion.svg>
            <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.95 }}
                transition={{ duration: 0.3, ease: "easeInOut" }}
                className="fixed z-[9999] pointer-events-none"
                style={{
                    left: highlightRect.left,
                    top: highlightRect.top,
                    width: highlightRect.width,
                    height: highlightRect.height,
                    border: '3px solid hsl(var(--brand))',
                    borderRadius: '8px',
                    boxShadow: '0 0 0 4px rgba(var(--brand-rgb, 59 130 246), 0.1), 0 0 20px hsl(var(--brand))',
                }}
            />
        </>
    );
};
