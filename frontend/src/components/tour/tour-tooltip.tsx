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

import { Button, Card } from '@clidey/ux';
import { motion } from 'framer-motion';
import type { FC } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from '../../hooks/use-translation';
import { useAppSelector } from '../../store/hooks';

export type TooltipPosition = 'top' | 'bottom' | 'left' | 'right' | 'center';

interface TourTooltipProps {
    targetElement: HTMLElement | null;
    title: string;
    description: string;
    position?: TooltipPosition;
    currentStep: number;
    totalSteps: number;
    onNext: () => void;
    onPrev: () => void;
    onSkip: () => void;
    isFirstStep: boolean;
    isLastStep: boolean;
}

export const TourTooltip: FC<TourTooltipProps> = ({
    targetElement,
    title,
    description,
    position = 'right',
    currentStep,
    totalSteps,
    onNext,
    onPrev,
    onSkip,
    isFirstStep,
    isLastStep,
}) => {
    const { t } = useTranslation('components/tour');
    const disableAnimations = useAppSelector(state => state.settings.disableAnimations);
    const [tooltipStyle, setTooltipStyle] = useState<React.CSSProperties>({});

    useEffect(() => {
        if (!targetElement) {
            setTooltipStyle({
                position: 'fixed',
                top: '50%',
                left: '50%',
                transform: 'translate(-50%, -50%)',
            });
            return;
        }

        const updatePosition = () => {
            const navItem = targetElement.closest('[data-slot="sidebar-menu-item"]');
            const rect = (navItem ?? targetElement).getBoundingClientRect();
            const tooltipWidth = Math.min(320, window.innerWidth - 32);
            const tooltipHeight = 240;
            const gap = 12;
            let left = rect.right + gap;
            let top = rect.top;

            switch (position) {
                case 'right':
                    left = Math.max(left, (targetElement.closest('[data-slot="sidebar-container"]')?.getBoundingClientRect().right ?? left) + 8);
                    top = rect.top + rect.height / 2 - 28;
                    break;
                case 'left':
                    left = rect.left - tooltipWidth - gap;
                    break;
                case 'bottom':
                    left = rect.left + (rect.width - tooltipWidth) / 2;
                    top = rect.bottom + gap;
                    break;
                case 'top':
                    left = rect.left + (rect.width - tooltipWidth) / 2;
                    top = rect.top - tooltipHeight - gap;
                    break;
                case 'center':
                default:
                    setTooltipStyle({
                        position: 'fixed',
                        left: '50%',
                        top: '50%',
                        transform: 'translate(-50%, -50%)',
                    });
                    return;
            }

            setTooltipStyle({
                position: 'fixed',
                left: Math.max(16, Math.min(left, window.innerWidth - tooltipWidth - 16)),
                top: Math.max(16, Math.min(top, window.innerHeight - tooltipHeight - 16)),
            });
        };

        updatePosition();
        window.addEventListener('resize', updatePosition);
        window.addEventListener('scroll', updatePosition, true);

        return () => {
            window.removeEventListener('resize', updatePosition);
            window.removeEventListener('scroll', updatePosition, true);
        };
    }, [targetElement, position]);

    return (
        <motion.div
            {...(disableAnimations ? {} : {
                initial: { opacity: 0, scale: 0.9, y: 10 },
                animate: { opacity: 1, scale: 1, y: 0 },
                exit: { opacity: 0, scale: 0.9, y: 10 },
                transition: { duration: 0.3, ease: "easeInOut" }
            })}
            style={tooltipStyle}
            className="tour-tooltip z-[10000]"
            data-position={position}
            data-testid="tour-tooltip"
        >
            <Card className="tour-tooltip-card">
                <p className="tour-tooltip-step">{t('tourStep', { current: currentStep, total: totalSteps })}</p>
                <h3>{title}</h3>
                <p className="tour-tooltip-description">{description}</p>
                <div className="tour-tooltip-footer">
                    <div className="tour-tooltip-dots" aria-hidden="true">
                        {Array.from({ length: totalSteps }, (_, step) => <span key={step} className={step === currentStep - 1 ? 'is-current' : ''} />)}
                    </div>
                    <div className="tour-tooltip-actions">
                        <Button onClick={onSkip} variant="ghost" size="sm" data-testid="tour-skip-button">{t('skip')}</Button>
                        {!isFirstStep && <Button onClick={onPrev} variant="outline" size="sm" data-testid="tour-prev-button">{t('back')}</Button>}
                        <Button onClick={onNext} size="sm" data-testid="tour-next-button">{isLastStep ? t('finish') : t('next')}<span className="tour-next-icon" aria-hidden="true" /></Button>
                    </div>
                </div>
            </Card>
        </motion.div>
    );
};
