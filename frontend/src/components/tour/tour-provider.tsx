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

import { FC, ReactNode, useCallback, useEffect } from 'react';
import { useAppDispatch, useAppSelector } from '../../store/hooks';
import { TourActions } from '../../store/tour';
import { sampleDatabaseTour } from '../../config/tour-config';
import { Tour } from './tour';
import { markOnboardingComplete } from '../../utils/onboarding';
import { featureFlags } from '../../config/features';

interface TourProviderProps {
    children: ReactNode;
}

export const TourProvider: FC<TourProviderProps> = ({ children }) => {
    const dispatch = useAppDispatch();
    const tourState = useAppSelector(state => state.tour);

    useEffect(() => {
        if (tourState.shouldStartOnLoad && tourState.tourId && featureFlags.autoStartTourOnLogin) {
            setTimeout(() => {
                if (tourState.tourId) {
                    dispatch(TourActions.startTour(tourState.tourId));
                }
            }, 1000);
        }
    }, [tourState.shouldStartOnLoad, tourState.tourId, dispatch]);

    const handleComplete = useCallback(() => {
        markOnboardingComplete();
        dispatch(TourActions.stopTour());
    }, [dispatch]);

    const handleSkip = useCallback(() => {
        markOnboardingComplete();
        dispatch(TourActions.stopTour());
    }, [dispatch]);

    const getTourConfig = useCallback(() => {
        switch (tourState.tourId) {
            case 'sample-database-tour':
                return sampleDatabaseTour;
            default:
                return null;
        }
    }, [tourState.tourId]);

    const config = getTourConfig();

    return (
        <>
            {children}
            {config && (
                <Tour
                    config={config}
                    isActive={tourState.isActive}
                    onComplete={handleComplete}
                    onSkip={handleSkip}
                />
            )}
        </>
    );
};
