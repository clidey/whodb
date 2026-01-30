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

import { Button, cn, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@clidey/ux';
import { useTranslation } from '@/hooks/use-translation';
import { useAppSelector } from '@/store/hooks';
import { useNavigate } from 'react-router-dom';
import { PublicRoutes } from '@/config/routes';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { DatabaseType, useLoginWithProfileMutation } from '@graphql';
import { useState } from 'react';

/**
 * ServerDownOverlay displays when the backend server is unreachable.
 * Shows a reconnection message with a spinner.
 * Only shows when user is logged in.
 */
export const ServerDownOverlay = () => {
    const { t } = useTranslation('components/health-overlay');
    const serverStatus = useAppSelector(state => state.health.serverStatus);
    const authStatus = useAppSelector(state => state.auth.status);

    // Only show overlay if user is logged in AND status is explicitly 'error'
    // Don't show for 'unknown' status (haven't checked yet)
    const shouldShow = authStatus === 'logged-in' && serverStatus === 'error';

    if (!shouldShow) {
        return null;
    }

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div
                className={cn(
                    'w-full max-w-md rounded-lg border border-destructive/50 bg-background p-6 shadow-2xl',
                    'animate-in fade-in zoom-in-95 duration-300'
                )}
            >
                <div className="flex flex-col items-center gap-4 text-center">
                    <XCircleIcon className="h-12 w-12 text-destructive" />
                    <div>
                        <h2 className="text-xl font-semibold text-foreground">
                            {t('serverDownTitle')}
                        </h2>
                        <p className="mt-2 text-sm text-muted-foreground">
                            {t('serverDownMessage')}
                        </p>
                    </div>
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                        <span>{t('serverDownRetrying')}</span>
                    </div>
                </div>
            </div>
        </div>
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
    const databaseStatus = useAppSelector(state => state.health.databaseStatus);
    const serverStatus = useAppSelector(state => state.health.serverStatus);
    const authStatus = useAppSelector(state => state.auth.status);
    const currentProfile = useAppSelector(state => state.auth.current);
    const allProfiles = useAppSelector(state => state.auth.profiles);

    const [selectedProfileId, setSelectedProfileId] = useState<string>('');
    const [loginWithProfile, { loading }] = useLoginWithProfileMutation();

    // Only show database overlay if:
    // - User is logged in
    // - Server is healthy
    // - Database is explicitly in 'error' state (not 'unavailable' or 'unknown')
    const shouldShow = authStatus === 'logged-in' &&
        serverStatus === 'healthy' &&
        databaseStatus === 'error';

    if (!shouldShow) {
        return null;
    }

    // Get profiles excluding the current one
    const otherProfiles = allProfiles.filter(p => p.Id !== currentProfile?.Id);
    const hasOtherProfiles = otherProfiles.length > 0;

    const handleSwitchProfile = async () => {
        if (!selectedProfileId) return;

        const profile = otherProfiles.find(p => p.Id === selectedProfileId);
        if (!profile) return;

        try {
            await loginWithProfile({
                variables: {
                    profile: {
                        Id: profile.Id,
                        Type: profile.Type as DatabaseType,
                        Database: profile.Database,
                    }
                }
            });

            // Reload page to refresh all data with new connection
            window.location.reload();
        } catch (error) {
            console.error('Failed to switch profile:', error);
        }
    };

    const handleLogout = () => {
        navigate(PublicRoutes.Login.path);
    };

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div
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
                        <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                        <span>{t('databaseDownRetrying')}</span>
                    </div>

                    {hasOtherProfiles ? (
                        <div className="flex flex-col gap-3 mt-4">
                            <div className="space-y-2">
                                <label className="text-sm font-medium">{t('selectProfile')}</label>
                                <Select value={selectedProfileId} onValueChange={setSelectedProfileId}>
                                    <SelectTrigger className="w-full">
                                        <SelectValue>
                                            {selectedProfileId
                                                ? otherProfiles.find(p => p.Id === selectedProfileId)?.Database + ' (' + otherProfiles.find(p => p.Id === selectedProfileId)?.Type + ')'
                                                : t('selectProfilePlaceholder')
                                            }
                                        </SelectValue>
                                    </SelectTrigger>
                                    <SelectContent>
                                        {otherProfiles.map((profile) => (
                                            <SelectItem key={profile.Id} value={profile.Id}>
                                                {profile.Database} ({profile.Type})
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
                                <Button variant="destructive" size="sm" onClick={handleLogout}>
                                    {t('logout')}
                                </Button>
                            </div>
                        </div>
                    ) : (
                        <div className="flex justify-end mt-4">
                            <Button variant="destructive" size="sm" onClick={handleLogout}>
                                {t('logout')}
                            </Button>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};
