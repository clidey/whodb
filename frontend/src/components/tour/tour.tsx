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

import { AnimatePresence } from 'framer-motion';
import { FC, useCallback, useEffect, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { TourConfig, TourStep } from './tour-step';
import { TourSpotlight } from './tour-spotlight';
import { TourTooltip } from './tour-tooltip';

interface TourProps {
    config: TourConfig;
    isActive: boolean;
    onComplete: () => void;
    onSkip: () => void;
}

export const Tour: FC<TourProps> = ({ config, isActive, onComplete, onSkip }) => {
    const navigate = useNavigate();
    const location = useLocation();
    const [currentStepIndex, setCurrentStepIndex] = useState(0);
    const [targetElement, setTargetElement] = useState<HTMLElement | null>(null);
    const [isTransitioning, setIsTransitioning] = useState(false);

    const currentStep = config.steps[currentStepIndex];
    const isFirstStep = currentStepIndex === 0;
    const isLastStep = currentStepIndex === config.steps.length - 1;

    const findTargetElement = useCallback((selector: string, callback: (element: HTMLElement | null) => void): void => {
        const maxAttempts = 30;
        let attempts = 0;

        const tryFind = () => {
            const element = document.querySelector(selector);
            if (element instanceof HTMLElement) {
                callback(element);
                return;
            }
            attempts++;
            if (attempts < maxAttempts) {
                setTimeout(() => tryFind(), 150);
            } else {
                callback(null);
            }
        };

        tryFind();
    }, []);

    const showStep = useCallback((step: TourStep) => {
        setIsTransitioning(true);
        setTargetElement(null);

        setTimeout(() => {
            const needsNavigation = step.path && location.pathname !== step.path;

            if (needsNavigation && step.path) {
                navigate(step.path);
            }

            const delay = needsNavigation ? 1000 : 300;

            setTimeout(() => {
                if (step.beforeShow) {
                    step.beforeShow();
                }

                setTimeout(() => {
                    findTargetElement(step.target, (element) => {
                        setTargetElement(element);
                        setIsTransitioning(false);

                        if (element) {
                            setTimeout(() => {
                                const rect = element.getBoundingClientRect();
                                const tooltipHeight = 400;
                                const viewportHeight = window.innerHeight;

                                const needsSpace = step.position === 'bottom' ?
                                    (rect.bottom + tooltipHeight + 40 > viewportHeight) :
                                    step.position === 'top' ?
                                    (rect.top - tooltipHeight - 40 < 0) :
                                    false;

                                element.scrollIntoView({
                                    behavior: 'smooth',
                                    block: needsSpace ? 'center' : 'nearest',
                                    inline: 'center',
                                });
                            }, 100);
                        }
                    });
                }, step.beforeShow ? 300 : 0);
            }, delay);
        }, 200);
    }, [findTargetElement, location.pathname, navigate]);

    useEffect(() => {
        if (isActive && currentStep) {
            showStep(currentStep);
        }
    }, [isActive, currentStep, showStep]);

    const handleNext = useCallback(() => {
        if (isLastStep) {
            onComplete();
        } else {
            setCurrentStepIndex(prev => prev + 1);
        }
    }, [isLastStep, onComplete]);

    const handlePrev = useCallback(() => {
        if (!isFirstStep) {
            setCurrentStepIndex(prev => prev - 1);
        }
    }, [isFirstStep]);

    const handleSkip = useCallback(() => {
        onSkip();
    }, [onSkip]);

    useEffect(() => {
        if (isActive) {
            document.body.style.overflow = 'hidden';
        } else {
            document.body.style.overflow = '';
        }

        return () => {
            document.body.style.overflow = '';
        };
    }, [isActive]);

    if (!isActive || !currentStep) {
        return null;
    }

    return (
        <>
            {!isTransitioning && targetElement && (
                <AnimatePresence mode="wait">
                    <TourSpotlight key={`spotlight-${currentStepIndex}`} targetElement={targetElement} />
                </AnimatePresence>
            )}
            <AnimatePresence mode="wait">
                {!isTransitioning && (
                    <TourTooltip
                        key={`tooltip-${currentStepIndex}`}
                        targetElement={targetElement}
                        title={currentStep.title}
                        description={currentStep.description}
                        icon={currentStep.icon}
                        position={currentStep.position}
                        currentStep={currentStepIndex + 1}
                        totalSteps={config.steps.length}
                        onNext={handleNext}
                        onPrev={handlePrev}
                        onSkip={handleSkip}
                        isFirstStep={isFirstStep}
                        isLastStep={isLastStep}
                    />
                )}
            </AnimatePresence>
        </>
    );
};
