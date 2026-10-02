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

import { motion } from 'framer-motion';
import type { FC} from 'react';
import { useEffect, useState } from 'react';
import { useAppSelector } from '../../store/hooks';

interface TourSpotlightProps {
    targetElement: HTMLElement | null;
    padding?: number;
}

export const TourSpotlight: FC<TourSpotlightProps> = ({ targetElement, padding = 8 }) => {
    const disableAnimations = useAppSelector(state => state.settings.disableAnimations);
    const [rect, setRect] = useState<DOMRect | null>(null);

    useEffect(() => {
        if (!targetElement) {
            setRect(null);
            return;
        }

        const updateRect = () => {
            const navItem = targetElement.closest('[data-slot="sidebar-menu-item"]');
            setRect((navItem ?? targetElement).getBoundingClientRect());
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
                {...(disableAnimations ? {} : {
                    initial: { opacity: 0 },
                    animate: { opacity: 1 },
                    exit: { opacity: 0 },
                    transition: { duration: 0.3, ease: "easeInOut" }
                })}
                className="tour-spotlight-overlay fixed inset-0 z-[9998] pointer-events-none"
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
                    fill="currentColor"
                    mask={`url(#tour-spotlight-mask-${rect.left}-${rect.top})`}
                />
            </motion.svg>
            <motion.div
                {...(disableAnimations ? {} : {
                    initial: { opacity: 0 },
                    animate: { opacity: 1 },
                    exit: { opacity: 0 },
                    transition: { duration: 0.3, ease: "easeInOut" }
                })}
                className="tour-spotlight-outline fixed z-[9999] pointer-events-none"
                style={{
                    left: highlightRect.left,
                    top: highlightRect.top,
                    width: highlightRect.width,
                    height: highlightRect.height,
                    border: '2px solid #416bd1',
                    borderRadius: '8px',
                }}
            />
        </>
    );
};
