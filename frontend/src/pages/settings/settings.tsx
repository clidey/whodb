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

import type {FC} from "react";
import { Suspense, useCallback, useEffect, useMemo} from "react";
import {InternalPage} from "../../components/page";
import {InternalRoutes, type IInternalRoute} from "../../config/routes";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {SettingsActions} from "../../store/settings";
import {useTranslation} from "@/hooks/use-translation";
import {
    Input,
    Label,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Switch,
    Tabs,
    TabsContent,
    TabsList,
    TabsTrigger,
} from "@clidey/ux";
import {optInUser, optOutUser, trackFrontendEvent} from "@/config/posthog";
import {type SupportedLanguage, SUPPORTED_LANGUAGES} from "@/utils/languages";
import {ExternalLink} from "../../utils/external-links";
import {usePageSize} from "../../hooks/use-page-size";
import {getComponent} from "../../config/component-registry";
import {trackOptionChanged} from "@/config/frontend-analytics";
import {useNavigate, useSearchParams} from 'react-router-dom';
import {WhoDBChatIcon} from '../../components/whodb-chat-icon';

export const SettingsPage: FC = () => {
    const {t} = useTranslation('pages/settings');
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const activeTab = searchParams.get('tab') ?? 'appearance';
    const dispatch = useAppDispatch();
    const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);
    const storageUnitView = useAppSelector(state => state.settings.storageUnitView);
    const fontSize = useAppSelector(state => state.settings.fontSize);
    const borderRadius = useAppSelector(state => state.settings.borderRadius);
    const spacing = useAppSelector(state => state.settings.spacing);
    const whereConditionMode = useAppSelector(state => state.settings.whereConditionMode);
    const formatDatesLocale = useAppSelector(state => state.settings.formatDatesLocale);
    const formatBooleansReadable = useAppSelector(state => state.settings.formatBooleansReadable);
    const defaultPageSize = useAppSelector(state => state.settings.defaultPageSize);
    const maxPageSize = useAppSelector(state => state.settings.maxPageSize);
    const language = useAppSelector(state => state.settings.language);
    const databaseSchemaTerminology = useAppSelector(state => state.settings.databaseSchemaTerminology);
    const disableAnimations = useAppSelector(state => state.settings.disableAnimations);

    const pageSizeOptions = useMemo(() => ({
        onPageSizeChange: (size: number) => {
            trackOptionChanged('default_page_size', size);
            dispatch(SettingsActions.setDefaultPageSize(size));
        },
        maxPageSize,
    }), [dispatch, maxPageSize]);

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
            void optInUser();
            void trackFrontendEvent('ui.telemetry_toggled', {enabled: true});
        } else {
            void trackFrontendEvent('ui.telemetry_toggled', {enabled: false});
            void optOutUser();
        }
        dispatch(SettingsActions.setMetricsEnabled(enabled));
    }, [dispatch]);

    const handleStorageUnitViewToggle = useCallback((view: 'list' | 'card') => {
        trackOptionChanged('storage_unit_view', view, {
            view_mode: view,
        });
        dispatch(SettingsActions.setStorageUnitView(view));
    }, [dispatch]);

    const handleFontSizeChange = useCallback((size: 'small' | 'medium' | 'large') => {
        trackOptionChanged('font_size', size);
        dispatch(SettingsActions.setFontSize(size));
    }, [dispatch]);

    const handleBorderRadiusChange = useCallback((radius: 'none' | 'small' | 'medium' | 'large') => {
        trackOptionChanged('border_radius', radius);
        dispatch(SettingsActions.setBorderRadius(radius));
    }, [dispatch]);

    const handleSpacingChange = useCallback((space: 'compact' | 'comfortable' | 'spacious') => {
        trackOptionChanged('spacing', space);
        dispatch(SettingsActions.setSpacing(space));
    }, [dispatch]);

    const handleWhereConditionModeChange = useCallback((mode: 'popover' | 'sheet') => {
        trackOptionChanged('where_condition_mode', mode);
        dispatch(SettingsActions.setWhereConditionMode(mode));
    }, [dispatch]);

    const handleLanguageChange = useCallback((lang: SupportedLanguage) => {
        trackOptionChanged('language', lang);
        dispatch(SettingsActions.setLanguage(lang));
    }, [dispatch]);

    const handleDatabaseSchemaTerminologyChange = useCallback((terminology: 'database' | 'schema') => {
        trackOptionChanged('database_schema_terminology', terminology);
        dispatch(SettingsActions.setDatabaseSchemaTerminology(terminology));
    }, [dispatch]);

    const handleDisableAnimationsToggle = useCallback((disabled: boolean) => {
        trackOptionChanged('disable_animations', disabled);
        dispatch(SettingsActions.setDisableAnimations(disabled));
    }, [dispatch]);

    const handleFormatDatesLocaleToggle = useCallback((enabled: boolean) => {
        trackOptionChanged('format_dates_locale', enabled);
        dispatch(SettingsActions.setFormatDatesLocale(enabled));
    }, [dispatch]);

    const handleFormatBooleansReadableToggle = useCallback((enabled: boolean) => {
        trackOptionChanged('format_booleans_readable', enabled);
        dispatch(SettingsActions.setFormatBooleansReadable(enabled));
    }, [dispatch]);

    const handleTabChange = useCallback((tab: string) => {
        setSearchParams({tab});
        trackOptionChanged('settings_tab', tab, {
            tab,
        });
    }, [setSearchParams]);

    const hasIntegrations = !!getComponent('bridge-driver-panel');

    return (
        <InternalPage routes={[InternalRoutes.Settings as IInternalRoute]}>
            <div className="ce-settings-page w-full">
                <Tabs value={activeTab} className="ce-settings-tabs" onValueChange={handleTabChange}>
                    <TabsList className="ce-settings-rail">
                        <div className="ce-settings-rail-heading"><h1>{t('settingsTitle')}</h1><p>{t('savedInBrowser')}</p></div>
                        <TabsTrigger value="appearance"><WhoDBChatIcon name="grid" />{t('tabAppearance')}</TabsTrigger>
                        <TabsTrigger value="behavior"><WhoDBChatIcon name="sliders" />{t('tabBehavior')}</TabsTrigger>
                        {hasIntegrations && <TabsTrigger value="integrations">{t('tabIntegrations')}</TabsTrigger>}
                        <TabsTrigger value="privacy"><WhoDBChatIcon name="secret" />{t('tabPrivacy')}</TabsTrigger>
                        <button type="button" onClick={() => void navigate(InternalRoutes.ContactUs?.path ?? '/contact-us')}><WhoDBChatIcon name="mail" />{t('contact')}</button>
                    </TabsList>

                    <TabsContent value="appearance" className="ce-settings-panel">
                        <div className="ce-settings-panel-heading"><h2>{t('tabAppearance')}</h2><p>{t('appearanceDescription')}</p></div>
                        <p className="ce-settings-section-label">{t('appearanceSection')}</p>
                        <div className="flex justify-between">
                            <div className="ce-settings-row-copy"><Label htmlFor="storage-unit-view">{t('tableView')}</Label><p>{t('tableViewDescription')}</p></div>
                            <Select value={storageUnitView} onValueChange={handleStorageUnitViewToggle}>
                                <SelectTrigger id="storage-unit-view" className="w-[165px]">
                                    <SelectValue placeholder={t('selectView')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="list" data-value="list">{t('list')}</SelectItem>
                                    <SelectItem value="card" data-value="card">{t('cards')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <div className="ce-settings-row-copy"><Label htmlFor="font-size">{t('textSize')}</Label><p>{t('textSizeDescription')}</p></div>
                            <Select value={fontSize} onValueChange={handleFontSizeChange}>
                                <SelectTrigger id="font-size" className="w-[165px]">
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
                            <div className="ce-settings-row-copy"><Label htmlFor="spacing">{t('density')}</Label><p>{t('densityDescription')}</p></div>
                            <Select value={spacing} onValueChange={handleSpacingChange}>
                                <SelectTrigger id="spacing" className="w-[165px]">
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
                            <div className="ce-settings-row-copy"><Label htmlFor="settings-animations">{t('animations')}</Label><p>{t('animationsDescription')}</p></div>
                            <Switch id="settings-animations" checked={!disableAnimations} onCheckedChange={enabled => { handleDisableAnimationsToggle(!enabled); }}/>
                        </div>
                        <p className="ce-settings-section-label">{t('dataDisplayTitle')}</p>
                        <div className="flex justify-between">
                            <div className="ce-settings-row-copy"><Label htmlFor="settings-local-dates">{t('localDates')}</Label><p>{t('localDatesDescription')}</p></div>
                            <Switch id="settings-local-dates" checked={formatDatesLocale} onCheckedChange={handleFormatDatesLocaleToggle}/>
                        </div>
                        <div className="flex justify-between">
                            <div className="ce-settings-row-copy"><Label htmlFor="settings-readable-booleans">{t('readableBooleans')}</Label><p>{t('readableBooleansDescription')}</p></div>
                            <Switch id="settings-readable-booleans" checked={formatBooleansReadable} onCheckedChange={handleFormatBooleansReadableToggle}/>
                        </div>
                    </TabsContent>

                    <TabsContent value="behavior" className="ce-settings-panel">
                        <div className="ce-settings-panel-heading"><h2>{t('tabBehavior')}</h2><p>{t('behaviorDescription')}</p></div>
                        <p className="ce-settings-section-label">{t('gridSection')}</p>
                        <div className="flex justify-between">
                            <div className="ce-settings-row-copy"><Label htmlFor="where-condition-mode">{t('whereConditionMode')}</Label><p>{t('whereConditionModeDescription')}</p></div>
                            <Select value={whereConditionMode} onValueChange={handleWhereConditionModeChange}>
                                <SelectTrigger id="where-condition-mode" className="w-[165px]">
                                    <SelectValue placeholder={t('selectMode')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="popover" data-value="popover">{t('popover')}</SelectItem>
                                    <SelectItem value="sheet" data-value="sheet">{t('sheet')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <div className="ce-settings-row-copy"><Label htmlFor="default-page-size">{t('defaultPageSize')}</Label><p>{t('defaultPageSizeDescription')}</p></div>
                            <div className="flex gap-2">
                                <Select
                                    value={isCustomPageSize ? "custom" : pageSizeString}
                                    onValueChange={handleDefaultPageSizeChange}
                                >
                                    <SelectTrigger id="default-page-size" className="w-[165px]">
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
                                        onChange={(e) => { setCustomPageSizeInput(e.target.value); }}
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
                        <p className="ce-settings-section-label">{t('generalSection')}</p>
                        <div className="flex justify-between">
                            <div className="ce-settings-row-copy"><Label htmlFor="border-radius">{t('cornerRadius')}</Label><p>{t('cornerRadiusDescription')}</p></div>
                            <Select value={borderRadius} onValueChange={handleBorderRadiusChange}>
                                <SelectTrigger id="border-radius" className="w-[165px]">
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
                            <div className="ce-settings-row-copy"><Label htmlFor="database-schema-terminology">{t('databaseSchemaTerminology')}</Label><p>{t('databaseSchemaTerminologyDescription')}</p></div>
                            <Select value={databaseSchemaTerminology} onValueChange={handleDatabaseSchemaTerminologyChange}>
                                <SelectTrigger id="database-schema-terminology" className="w-[165px]">
                                    <SelectValue placeholder={t('selectDatabaseSchemaTerminology')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="database" data-value="database">{t('database')}</SelectItem>
                                    <SelectItem value="schema" data-value="schema">{t('databaseSchemaTerminologySchema')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <div className="ce-settings-row-copy"><Label htmlFor="language">{t('language')}</Label><p>{t('languageDescription')}</p></div>
                            <Select value={language} onValueChange={handleLanguageChange}>
                                <SelectTrigger id="language" className="w-[200px]">
                                    <SelectValue placeholder={t('selectLanguage')} />
                                </SelectTrigger>
                                <SelectContent>
                                    {Object.entries(SUPPORTED_LANGUAGES).map(([code, name]) => (
                                        <SelectItem key={code} value={code} data-value={code}>{name}</SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                    </TabsContent>

                    {hasIntegrations && (
                        <TabsContent value="integrations" className="ce-settings-panel">
                            {(() => {
                                const BridgeDriverPanel = getComponent('bridge-driver-panel');
                                if (!BridgeDriverPanel) return null;
                                return (
                                    <Suspense fallback={null}>
                                        <BridgeDriverPanel />
                                    </Suspense>
                                );
                            })()}
                        </TabsContent>
                    )}

                    <TabsContent value="privacy" className="ce-settings-panel">
                        <div className="ce-settings-panel-heading"><h2>{t('tabPrivacy')}</h2><p>{t('privacyDescription')}</p></div>
                        <p className="ce-settings-section-label">{t('usageDataSection')}</p>
                        <div className="ce-settings-privacy-card">
                            <div className="flex justify-between">
                                <div className="ce-settings-row-copy"><Label htmlFor="settings-metrics">{t('anonymousUsageData')}</Label><p>{t('anonymousUsageDataDescription')}</p></div>
                                <Switch id="settings-metrics" checked={metricsEnabled} onCheckedChange={handleMetricsToggle}/>
                            </div>
                        </div>
                        <p className="ce-settings-section-label">{t('moreSection')}</p>
                        <div className="ce-settings-privacy-card">
                            <div className="flex justify-between">
                                <div className="ce-settings-row-copy"><span className="ce-settings-row-title">{t('privacyPolicy')}</span><p>{t('privacyPolicyDescription')}</p></div>
                                <ExternalLink href="https://whodb.com/privacy" className="ce-settings-link-button">{t('open')}<WhoDBChatIcon name="chevron-right" /></ExternalLink>
                            </div>
                        </div>
                    </TabsContent>
                </Tabs>
            </div>
        </InternalPage>
    );
}
