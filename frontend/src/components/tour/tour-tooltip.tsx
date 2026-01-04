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

import { Badge, Button, Card } from '@clidey/ux';
import { motion } from 'framer-motion';
import { FC, ReactElement, useEffect, useState } from 'react';
import { CheckCircleIcon, ChevronRightIcon, XMarkIcon } from '../heroicons';
import { useTranslation } from '../../hooks/use-translation';
import { useAppSelector } from '../../store/hooks';

export type TooltipPosition = 'top' | 'bottom' | 'left' | 'right' | 'center';

interface TourTooltipProps {
    targetElement: HTMLElement | null;
    title: string;
    description: string;
    icon?: ReactElement;
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
    icon,
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
            const rect = targetElement.getBoundingClientRect();
            const tooltipWidth = 400;
            const tooltipHeight = 300;
            const gap = 20;

            let style: React.CSSProperties = {
                position: 'fixed',
            };

            switch (position) {
                case 'right':
                    style = {
                        ...style,
                        left: rect.right + gap,
                        top: rect.top + rect.height / 2,
                        transform: 'translateY(-50%)',
                    };
                    break;
                case 'left':
                    style = {
                        ...style,
                        right: window.innerWidth - rect.left + gap,
                        top: rect.top + rect.height / 2,
                        transform: 'translateY(-50%)',
                    };
                    break;
                case 'bottom':
                    style = {
                        ...style,
                        left: rect.left + rect.width / 2,
                        top: rect.bottom + gap,
                        transform: 'translateX(-50%)',
                    };
                    break;
                case 'top':
                    style = {
                        ...style,
                        left: rect.left + rect.width / 2,
                        bottom: window.innerHeight - rect.top + gap,
                        transform: 'translateX(-50%)',
                    };
                    break;
                case 'center':
                default:
                    style = {
                        ...style,
                        left: '50%',
                        top: '50%',
                        transform: 'translate(-50%, -50%)',
                    };
            }

            if (style.left && typeof style.left === 'number') {
                if (style.left + tooltipWidth > window.innerWidth) {
                    style.left = window.innerWidth - tooltipWidth - 20;
                }
                if (style.left < 20) {
                    style.left = 20;
                }
            }

            if (style.top && typeof style.top === 'number') {
                // TODO: Compensate for unknown offset in tooltip positioning
                style.top -= 25;
                if (style.top + tooltipHeight > window.innerHeight) {
                    style.top = window.innerHeight - tooltipHeight - 20;
                }
                if (style.top < 20) {
                    style.top = 20;
                }
            }

            setTooltipStyle(style);
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
            className="z-[10000] w-[400px]"
            data-testid="tour-tooltip"
        >
            <Card className="flex flex-col gap-4 p-6 shadow-2xl border-2 border-brand/30">
                <button
                    onClick={onSkip}
                    className="absolute top-4 right-4 text-muted-foreground hover:text-foreground transition-colors"
                    aria-label="Close tour"
                    data-testid="tour-skip-button"
                >
                    <XMarkIcon className="w-5 h-5" />
                </button>

                {icon && (
                    <div className="h-12 w-12 rounded-xl flex justify-center items-center bg-gradient-to-br from-brand to-brand/80 shadow-lg">
                        {icon}
                    </div>
                )}

                <div className="flex flex-col gap-2">
                    <h3 className="text-xl font-semibold text-foreground pr-8">
                        {title}
                    </h3>
                    <p className="text-sm text-muted-foreground leading-relaxed">
                        {description}
                    </p>
                </div>

                <div className="flex items-center justify-between pt-4 border-t">
                    <div className="flex items-center gap-2">
                        <Badge variant="secondary" className="text-xs">
                            {currentStep} / {totalSteps}
                        </Badge>
                        <div className="flex gap-1">
                            {Array.from({ length: totalSteps }).map((_, i) => (
                                <div
                                    key={i}
                                    className={`h-1.5 rounded-full transition-all ${
                                        i === currentStep - 1
                                            ? 'w-6 bg-brand'
                                            : 'w-1.5 bg-muted'
                                    }`}
                                />
                            ))}
                        </div>
                    </div>

                    <div className="flex gap-2">
                        {!isFirstStep && (
                            <Button onClick={onPrev} variant="outline" size="sm" data-testid="tour-prev-button">
                                <ChevronRightIcon className="w-4 h-4 rotate-180" />
                                {t('back')}
                            </Button>
                        )}
                        <Button onClick={onNext} size="sm" data-testid="tour-next-button">
                            {isLastStep ? (
                                <>
                                    <CheckCircleIcon className="w-4 h-4" />
                                    {t('finish')}
                                </>
                            ) : (
                                <>
                                    {t('next')}
                                    <ChevronRightIcon className="w-4 h-4" />
                                </>
                            )}
                        </Button>
                    </div>
                </div>
            </Card>
        </motion.div>
    );
};
