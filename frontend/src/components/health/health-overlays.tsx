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

import { Button, Card, CardContent, CardFooter, cn, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, toast } from '@clidey/ux';
import { Spinner } from '@/components/loading';
import { healthCheckService } from '@/services/health-check';
import { createPortal } from 'react-dom';
import { useTranslation } from '@/hooks/use-translation';
import { useAppSelector } from '@/store/hooks';
import { useNavigate, useLocation } from 'react-router-dom';
import { PublicRoutes } from '@/config/routes';
import { performLogout } from '@/config/logout-handler';
import { ArrowPathIcon, SignalSlashIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { useEffect, useState } from 'react';
import type { LocalLoginProfile } from '@/store/auth';
import { useProfileSwitch } from '@/hooks/use-profile-switch';
import { getAppName } from '@/config/features';

/**
 * Generate a display label for a profile in the health overlay.
 * Handles missing data gracefully.
 */
function getProfileLabel(profile: LocalLoginProfile): string {
    // Prefer database name if available
    if (profile.Database) {
        return profile.Type
            ? `${profile.Database} (${profile.Type})`
            : profile.Database;
    }
    // Fall back to hostname for databases like Redis
    if (profile.Hostname) {
        return profile.Type
            ? `${profile.Hostname} (${profile.Type})`
            : profile.Hostname;
    }
    // Last resort: use ID or Type
    return profile.Id || profile.Type || 'Unknown';
}

/**
 * ServerDownOverlay displays when the backend server is unreachable.
 * Shows reconnection status and offers an immediate retry.
 * On login page: shows when login fails with network error
 * When logged in: shows when health check detects server down
 */
export const ServerDownOverlay = () => {
    const { t } = useTranslation('components/health-overlay');
    const appName = getAppName();
    const serverStatus = useAppSelector(state => state.health.serverStatus);
    const [secondsUntilRetry, setSecondsUntilRetry] = useState<number | null>(null);
    const [isRetrying, setIsRetrying] = useState(false);

    // Show overlay whenever server status is explicitly 'error'
    // This happens when:
    // 1. Login fails with network error (login page)
    // 2. Health check detects server down (when logged in)
    const shouldShow = serverStatus === 'error';

    useEffect(() => {
        if (!shouldShow) return;

        const updateRetryTime = () => {
            setSecondsUntilRetry(healthCheckService.getSecondsUntilNextCheck());
        };
        updateRetryTime();
        const intervalId = setInterval(updateRetryTime, 250);
        return () => {
            clearInterval(intervalId);
        };
    }, [shouldShow]);

    const handleRetry = async () => {
        setIsRetrying(true);
        try {
            await healthCheckService.forceCheck();
        } finally {
            setIsRetrying(false);
        }
    };

    if (!shouldShow) {
        return null;
    }

    return createPortal(
        <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-slate-950/60 px-4 backdrop-blur-[3px]">
            <Card
                data-testid="health-overlay"
                role="alertdialog"
                aria-modal="true"
                aria-labelledby="server-down-title"
                aria-describedby="server-down-message"
                className={cn(
                    'w-full max-w-sm gap-0 overflow-hidden rounded-xl border-border bg-card py-0 shadow-2xl',
                    'animate-in fade-in zoom-in-95 duration-300'
                )}
            >
                <CardContent className="px-5 pb-4 pt-5">
                    <div className="flex size-9 items-center justify-center rounded-lg bg-destructive/10 text-destructive">
                        <SignalSlashIcon className="size-5" aria-hidden="true" />
                    </div>
                    <h2 id="server-down-title" className="mt-3 text-lg font-semibold text-foreground">
                        {t('serverDownTitle')}
                    </h2>
                    <p id="server-down-message" className="mt-1 text-sm text-muted-foreground">
                        {t('serverDownMessage', { appName })}
                    </p>
                    <div className="mt-4 flex items-center gap-2 text-xs text-muted-foreground">
                        <Spinner className="size-5" aria-hidden="true" />
                        <span className="font-medium text-foreground">{t('reconnecting')}</span>
                        {secondsUntilRetry !== null && (
                            <span className="ml-auto tabular-nums">
                                {t('retryingIn', { seconds: secondsUntilRetry })}
                            </span>
                        )}
                    </div>
                </CardContent>
                <CardFooter className="justify-end border-t border-border px-5 pb-3 !pt-3">
                    <Button size="sm" onClick={() => void handleRetry()} disabled={isRetrying}>
                        <ArrowPathIcon className="size-3.5" aria-hidden="true" />
                        {t('retryNow')}
                    </Button>
                </CardFooter>
            </Card>
        </div>,
        document.body
    );
};

/**
 * DatabaseDownOverlay displays when the database connection is lost.
 * Provides options to switch profiles or logout.
 * Only shows when user is logged in.
 */
export const DatabaseDownOverlay = () => {
    const { t } = useTranslation('components/health-overlay');
    const navigate = useNavigate();
    const location = useLocation();
    const databaseStatus = useAppSelector(state => state.health.databaseStatus);
    const serverStatus = useAppSelector(state => state.health.serverStatus);
    const authStatus = useAppSelector(state => state.auth.status);
    const currentProfileId = useAppSelector(state => state.auth.current?.Id);
    const allProfiles = useAppSelector(state => state.auth.profiles);
    const isEmbedded = useAppSelector(state => state.auth.isEmbedded);

    const [selectedProfileId, setSelectedProfileId] = useState<string>('');
    const [isSwitching, setIsSwitching] = useState(false);

    const { switchProfile, loading } = useProfileSwitch({
        onSuccess: () => {
            toast.success(t('switchSuccessful'));
        },
        onError: () => {
            setIsSwitching(false);
        },
        errorMessage: t('switchFailed'),
    });

    // Only show database overlay if:
    // - User is logged in (don't show when logged out)
    // - Not on the login page (prevents showing during logout/redirect)
    // - Server is healthy
    // - Database is explicitly in 'error' state (not 'unavailable' or 'unknown')
    // - Not currently switching profiles (hide during switch)
    const isOnLoginPage = location.pathname === PublicRoutes.Login.path;
    const shouldShow = authStatus === 'logged-in' &&
        !isOnLoginPage &&
        serverStatus === 'healthy' &&
        databaseStatus === 'error' &&
        !isSwitching;

    if (!shouldShow) {
        return null;
    }

    // Get profiles excluding the current one
    const otherProfiles = allProfiles.filter(p => p.Id !== currentProfileId);
    const hasOtherProfiles = otherProfiles.length > 0;

    const handleSwitchProfile = () => {
        if (!selectedProfileId) return;

        const profile = otherProfiles.find(p => p.Id === selectedProfileId);
        if (!profile) return;

        // Hide the dialog immediately when switching starts
        setIsSwitching(true);
        void switchProfile(profile);
    };

    const handleLogout = () => {
        performLogout(navigate);
    };

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div
                data-health-overlay
                className={cn(
                    'w-full max-w-md rounded-lg border border-destructive/50 bg-background p-6 shadow-2xl',
                    'animate-in fade-in zoom-in-95 duration-300'
                )}
            >
                <div className="flex flex-col gap-4">
                    <div className="flex items-start gap-3">
                        <XCircleIcon className="h-6 w-6 flex-shrink-0 text-destructive" />
                        <div>
                            <h2 className="text-lg font-semibold text-foreground">
                                {t('databaseDownTitle')}
                            </h2>
                            <p className="mt-2 text-sm text-muted-foreground">
                                {t('databaseDownMessage')}
                            </p>
                        </div>
                    </div>
                    <div className="flex items-center gap-2 text-sm text-muted-foreground pl-9">
                        <Spinner className="size-4" />
                        <span>{t('reconnecting')}</span>
                    </div>

                    {hasOtherProfiles ? (
                        <div className="flex flex-col gap-3 mt-4">
                            <div className="space-y-2">
                                <label className="text-sm font-medium">{t('selectProfile')}</label>
                                <Select value={selectedProfileId} onValueChange={setSelectedProfileId}>
                                    <SelectTrigger className="w-full">
                                        <SelectValue>
                                            {selectedProfileId
                                                ? (() => {
                                                    const profile = otherProfiles.find(p => p.Id === selectedProfileId);
                                                    return profile ? getProfileLabel(profile) : t('selectProfilePlaceholder');
                                                })()
                                                : t('selectProfilePlaceholder')
                                            }
                                        </SelectValue>
                                    </SelectTrigger>
                                    <SelectContent>
                                        {otherProfiles.map((profile) => (
                                            <SelectItem key={profile.Id} value={profile.Id}>
                                                {getProfileLabel(profile)}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </div>
                            <div className="flex gap-2 justify-end">
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={handleSwitchProfile}
                                    disabled={!selectedProfileId || loading}
                                >
                                    {loading ? t('switching') : t('switchProfile')}
                                </Button>
                                {!isEmbedded && (
                                    <Button variant="destructive" size="sm" onClick={handleLogout}>
                                        {t('logout')}
                                    </Button>
                                )}
                            </div>
                        </div>
                    ) : (
                        !isEmbedded && (
                            <div className="flex justify-end mt-4">
                                <Button variant="destructive" size="sm" onClick={handleLogout}>
                                    {t('logout')}
                                </Button>
                            </div>
                        )
                    )}
                </div>
            </div>
        </div>
    );
};
