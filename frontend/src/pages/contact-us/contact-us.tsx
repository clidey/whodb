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

import {Button} from "@clidey/ux";
import type {FC} from "react";
import {InternalPage} from "../../components/page";
import {InternalRoutes, type IInternalRoute} from "../../config/routes";
import {openExternalLink} from "../../utils/external-links";
import {useTranslation} from '@/hooks/use-translation';
import {useNavigate} from 'react-router-dom';
import {WhoDBChatIcon} from '../../components/whodb-chat-icon';

export const ContactUsPage: FC = () => {
    const { t } = useTranslation('pages/contact-us');
    const navigate = useNavigate();
    return (
        <InternalPage routes={[InternalRoutes.ContactUs as IInternalRoute]}>
            <div className="ce-settings-page w-full">
                <div className="ce-settings-tabs">
                    <nav className="ce-settings-rail" aria-label={t('settingsTitle')}>
                        <div className="ce-settings-rail-heading"><h1>{t('settingsTitle')}</h1><p>{t('savedInBrowser')}</p></div>
                        <button type="button" onClick={() => void navigate('/settings?tab=appearance')}><WhoDBChatIcon name="grid" />{t('appearance')}</button>
                        <button type="button" onClick={() => void navigate('/settings?tab=behavior')}><WhoDBChatIcon name="sliders" />{t('behavior')}</button>
                        <button type="button" onClick={() => void navigate('/settings?tab=privacy')}><WhoDBChatIcon name="secret" />{t('privacy')}</button>
                        <span className="is-active"><WhoDBChatIcon name="mail" />{t('contactUs')}</span>
                    </nav>
                    <div className="ce-settings-panel">
                        <div className="ce-settings-panel-heading"><h2>{t('contactUs')}</h2><p>{t('description')}</p></div>
                        <div className="ce-settings-contact-list">
                            <div><span className="ce-settings-contact-icon"><WhoDBChatIcon name="mail" /></span><div><strong>{t('emailTitle')}</strong><small>{t('emailAddress')} · {t('emailDescription')}</small></div><Button size="sm" onClick={() => { window.location.href = `mailto:${t('emailAddress')}`; }} data-testid="contact-email">{t('writeToUs')}</Button></div>
                            <div><span className="ce-settings-contact-icon"><WhoDBChatIcon name="globe" /></span><div><strong>{t('communityTitle')}</strong><small>{t('communityDescription')}</small></div><Button size="sm" variant="outline" data-testid="github-issue-button" onClick={(e) => { void openExternalLink('https://github.com/clidey/whodb/issues', e); }}>{t('submitIssue')}</Button></div>
                            <div><span className="ce-settings-contact-icon"><WhoDBChatIcon name="file-text" /></span><div><strong>{t('docsTitle')}</strong><small>{t('docsDescription')}</small></div><Button size="sm" variant="outline" onClick={(e) => { void openExternalLink('https://docs.whodb.com', e); }}>{t('readDocs')}</Button></div>
                        </div>
                        <p className="ce-settings-contact-note">{t('urgentNote')} <code>{t('urgentLabel')}</code> {t('urgentNoteEnd')}</p>
                    </div>
                </div>
            </div>
        </InternalPage>
    );
}
