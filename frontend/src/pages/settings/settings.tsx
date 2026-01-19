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

import {FC, useCallback, useEffect, useMemo} from "react";
import {InternalPage} from "../../components/page";
import {InternalRoutes} from "../../config/routes";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {SettingsActions} from "../../store/settings";
import {isEEMode} from "@/config/ee-imports";
import {useTranslation} from "@/hooks/use-translation";
import {
    Input,
    Label,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Separator,
    Switch,
} from "@clidey/ux";
import {optInUser, optOutUser, trackFrontendEvent} from "@/config/posthog";
import {ExternalLink} from "../../utils/external-links";
import {usePageSize} from "../../hooks/use-page-size";
import {AwsProvidersSection} from "../../components/aws";
import {useSettingsConfigQuery} from "@graphql";

export const SettingsPage: FC = () => {
    const {t} = useTranslation('pages/settings');
    const dispatch = useAppDispatch();
    const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);
    const storageUnitView = useAppSelector(state => state.settings.storageUnitView);
    const fontSize = useAppSelector(state => state.settings.fontSize);
    const borderRadius = useAppSelector(state => state.settings.borderRadius);
    const spacing = useAppSelector(state => state.settings.spacing);
    const whereConditionMode = useAppSelector(state => state.settings.whereConditionMode);
    const defaultPageSize = useAppSelector(state => state.settings.defaultPageSize);
    const language = useAppSelector(state => state.settings.language);
    const databaseSchemaTerminology = useAppSelector(state => state.settings.databaseSchemaTerminology);
    const disableAnimations = useAppSelector(state => state.settings.disableAnimations);

    // Check if cloud providers are enabled
    const { data: settingsData } = useSettingsConfigQuery();
    const cloudProvidersEnabled = settingsData?.SettingsConfig?.CloudProvidersEnabled ?? false;

    const pageSizeOptions = useMemo(() => ({
        onPageSizeChange: (size: number) => dispatch(SettingsActions.setDefaultPageSize(size)),
    }), [dispatch]);

    const {
        pageSizeString,
        isCustom: isCustomPageSize,
        customInput: customPageSizeInput,
        setCustomInput: setCustomPageSizeInput,
        handleSelectChange: handleDefaultPageSizeChange,
        handleCustomApply: handleCustomPageSizeApply,
    } = usePageSize(defaultPageSize, pageSizeOptions);

    useEffect(() => {
        void trackFrontendEvent('ui.settings_viewed');
    }, []);

    const handleMetricsToggle = useCallback((enabled: boolean) => {
        if (enabled) {
            optInUser();
            void trackFrontendEvent('ui.telemetry_toggled', {enabled: true});
        } else {
            void trackFrontendEvent('ui.telemetry_toggled', {enabled: false});
            optOutUser();
        }
        dispatch(SettingsActions.setMetricsEnabled(enabled));
    }, [dispatch, trackFrontendEvent]);

    const handleStorageUnitViewToggle = useCallback((view: 'list' | 'card') => {
        dispatch(SettingsActions.setStorageUnitView(view));
    }, [dispatch]);

    const handleFontSizeChange = useCallback((size: 'small' | 'medium' | 'large') => {
        dispatch(SettingsActions.setFontSize(size));
    }, [dispatch]);

    const handleBorderRadiusChange = useCallback((radius: 'none' | 'small' | 'medium' | 'large') => {
        dispatch(SettingsActions.setBorderRadius(radius));
    }, [dispatch]);

    const handleSpacingChange = useCallback((space: 'compact' | 'comfortable' | 'spacious') => {
        dispatch(SettingsActions.setSpacing(space));
    }, [dispatch]);

    const handleWhereConditionModeChange = useCallback((mode: 'popover' | 'sheet') => {
        dispatch(SettingsActions.setWhereConditionMode(mode));
    }, [dispatch]);

    const handleLanguageChange = useCallback((lang: 'en' | 'es') => {
        dispatch(SettingsActions.setLanguage(lang));
    }, [dispatch]);

    const handleDatabaseSchemaTerminologyChange = useCallback((terminology: 'database' | 'schema') => {
        dispatch(SettingsActions.setDatabaseSchemaTerminology(terminology));
    }, [dispatch]);

    const handleDisableAnimationsToggle = useCallback((disabled: boolean) => {
        dispatch(SettingsActions.setDisableAnimations(disabled));
    }, [dispatch]);

    return (
        <InternalPage routes={[InternalRoutes.Settings!]}>
            <div className="flex flex-col items-center w-full max-w-2xl mx-auto py-10 gap-8">
                <div className="w-full flex flex-col gap-0">
                    <div className="flex flex-col gap-2">
                        <p className="text-2xl font-bold flex items-center gap-2">
                            {t('telemetryTitle')}
                        </p>
                    </div>
                    <div className="flex flex-col gap-xl py-6">
                        {isEEMode ? (
                            <div className="flex flex-col gap-sm">
                                <h3 className="text-base">
                                    {t('eeNoTelemetry')}
                                </h3>
                            </div>
                        ) : (
                            <div className="flex flex-col gap-4">
                                <h3 className="text-base">
                                    {t('telemetryDescription')}&nbsp;
                                    {t('dataCollectionDetails')}&nbsp;
                                    <ExternalLink
                                    href={"https://clidey.com/privacy-policy"}
                                    className={"underline text-blue-500"}>{t('privacyPolicy')}</ExternalLink>.
                                    <br/>
                                    <br/>
                                    {t('posthogInfo')}&nbsp;
                                    {t('sensitiveDataInfo')}
                                    <br/>
                                    <br/>
                                    {t('contactUsInfo')}
                                </h3>
                                <br/>
                                <div className="flex justify-between">
                                    <Label>{metricsEnabled ? t('enableTelemetry') : t('disableTelemetry')}</Label>
                                    <Switch checked={metricsEnabled} onCheckedChange={handleMetricsToggle}/>
                                </div>
                                <Separator className="mt-4" />
                            </div>
                        )}
                        <div className="flex flex-col gap-sm mb-2">
                            <p className="text-lg font-bold">
                                {t('personalizeTitle')}
                            </p>
                            <p className="text-base">
                                {t('personalizeDescription')}
                            </p>
                        </div>
                        <div className="flex justify-between">
                            <Label>{t('storageUnitView')}</Label>
                            <Select value={storageUnitView} onValueChange={handleStorageUnitViewToggle}>
                                <SelectTrigger id="storage-unit-view" className="w-[135px]">
                                    <SelectValue placeholder={t('selectView')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="list" data-value="list">{t('list')}</SelectItem>
                                    <SelectItem value="card" data-value="card">{t('card')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>{t('fontSize')}</Label>
                            <Select value={fontSize} onValueChange={handleFontSizeChange}>
                                <SelectTrigger id="font-size" className="w-[135px]">
                                    <SelectValue placeholder={t('selectFontSize')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="small" data-value="small">{t('small')}</SelectItem>
                                    <SelectItem value="medium" data-value="medium">{t('medium')}</SelectItem>
                                    <SelectItem value="large" data-value="large">{t('large')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>{t('borderRadius')}</Label>
                            <Select value={borderRadius} onValueChange={handleBorderRadiusChange}>
                                <SelectTrigger id="border-radius" className="w-[135px]">
                                    <SelectValue placeholder={t('selectBorderRadius')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="none" data-value="none">{t('none')}</SelectItem>
                                    <SelectItem value="small" data-value="small">{t('small')}</SelectItem>
                                    <SelectItem value="medium" data-value="medium">{t('medium')}</SelectItem>
                                    <SelectItem value="large" data-value="large">{t('large')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>{t('spacing')}</Label>
                            <Select value={spacing} onValueChange={handleSpacingChange}>
                                <SelectTrigger id="spacing" className="w-[135px]">
                                    <SelectValue placeholder={t('selectSpacing')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="compact" data-value="compact">{t('compact')}</SelectItem>
                                    <SelectItem value="comfortable" data-value="comfortable">{t('comfortable')}</SelectItem>
                                    <SelectItem value="spacious" data-value="spacious">{t('spacious')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>{disableAnimations ? t('disableAnimationsEnabled') : t('disableAnimationsDisabled')}</Label>
                            <Switch checked={disableAnimations} onCheckedChange={handleDisableAnimationsToggle}/>
                        </div>
                        <div className="flex justify-between">
                            <Label>{t('whereConditionMode')}</Label>
                            <Select value={whereConditionMode} onValueChange={handleWhereConditionModeChange}>
                                <SelectTrigger id="where-condition-mode" className="w-[135px]">
                                    <SelectValue placeholder={t('selectMode')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="popover" data-value="popover">{t('popover')}</SelectItem>
                                    <SelectItem value="sheet" data-value="sheet">{t('sheet')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>{t('defaultPageSize')}</Label>
                            <div className="flex gap-2">
                                <Select
                                    value={isCustomPageSize ? "custom" : pageSizeString}
                                    onValueChange={handleDefaultPageSizeChange}
                                >
                                    <SelectTrigger id="default-page-size" className="w-[135px]">
                                        <SelectValue placeholder={t('selectPageSize')}/>
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="10" data-value="10">10</SelectItem>
                                        <SelectItem value="25" data-value="25">25</SelectItem>
                                        <SelectItem value="50" data-value="50">50</SelectItem>
                                        <SelectItem value="100" data-value="100">100</SelectItem>
                                        <SelectItem value="250" data-value="250">250</SelectItem>
                                        <SelectItem value="500" data-value="500">500</SelectItem>
                                        <SelectItem value="1000" data-value="1000">1000</SelectItem>
                                        <SelectItem value="custom" data-value="custom">{t('custom')}</SelectItem>
                                    </SelectContent>
                                </Select>
                                {isCustomPageSize && (
                                    <Input
                                        type="number"
                                        min={1}
                                        className="w-24"
                                        value={customPageSizeInput}
                                        onChange={(e) => setCustomPageSizeInput(e.target.value)}
                                        onBlur={handleCustomPageSizeApply}
                                        onKeyDown={(e) => {
                                            if (e.key === "Enter") {
                                                handleCustomPageSizeApply();
                                            }
                                        }}
                                    />
                                )}
                            </div>
                        </div>
                        <div className="flex justify-between">
                            <Label>{t('databaseSchemaTerminology')}</Label>
                            <Select value={databaseSchemaTerminology} onValueChange={handleDatabaseSchemaTerminologyChange}>
                                <SelectTrigger id="database-schema-terminology" className="w-[135px]">
                                    <SelectValue placeholder={t('selectDatabaseSchemaTerminology')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="database" data-value="database">{t('databaseSchemaTerminologyDatabase')}</SelectItem>
                                    <SelectItem value="schema" data-value="schema">{t('databaseSchemaTerminologySchema')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        {isEEMode && (
                            <div className="flex justify-between">
                                <Label>{t('language')}</Label>
                                <Select value={language} onValueChange={handleLanguageChange}>
                                    <SelectTrigger id="language" className="w-[135px]">
                                        <SelectValue placeholder={t('selectLanguage')} />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="en" data-value="en">English</SelectItem>
                                        <SelectItem value="es" data-value="es">Español</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>
                        )}
                        {cloudProvidersEnabled && (
                            <>
                                <Separator className="my-6" />
                                <AwsProvidersSection />
                            </>
                        )}
                    </div>
                </div>
            </div>
        </InternalPage>
    );
}
