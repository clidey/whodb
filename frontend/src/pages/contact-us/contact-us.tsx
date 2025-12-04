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

import {Badge, Button, Label, Separator} from "@clidey/ux";
import {ChatBubbleLeftRightIcon, EnvelopeIcon, GlobeAltIcon} from "../../components/heroicons";
import {FC} from "react";
import {InternalPage} from "../../components/page";
import {InternalRoutes} from "../../config/routes";
import {openExternalLink} from "../../utils/external-links";
import {useTranslation} from '@/hooks/use-translation';

export const ContactUsPage: FC = () => {
    const { t } = useTranslation('pages/contact-us');
    return (
        <InternalPage routes={[InternalRoutes.ContactUs!]}>
            <div className="flex flex-col items-center w-full max-w-2xl mx-auto py-10 gap-8">
                <div className="w-full flex flex-col gap-0">
                    <div className="flex flex-col gap-sm mb-4">
                        <div className="text-2xl font-bold flex items-center gap-2">
                            <EnvelopeIcon className="w-6 h-6"/>
                            {t('title')}
                        </div>
                        <p className="mt-2">{t('description')}</p>
                    </div>
                    <Separator/>
                    <div className="flex flex-col gap-xl py-6">
                        <div className="flex flex-col gap-2">
                            <Label className="text-lg font-semibold">{t('emailTitle')}</Label>
                            <Badge>
                                <a
                                    href={`mailto:${t('emailAddress')}`}
                                    className="transition-colors text-base font-medium"
                                    data-testid="contact-email"
                                >
                                    {t('emailAddress')}
                                </a>
                            </Badge>
                            <p className="text-sm">{t('emailDescription')}</p>
                        </div>
                        <Separator/>
                        <div className="flex flex-col gap-2">
                            <Label className="text-lg font-semibold">{t('communityTitle')}</Label>
                            <Button
                                variant="secondary"
                                className="w-fit gap-2"
                                data-testid="github-issue-button"
                                onClick={(e) => openExternalLink("https://github.com/clidey/whodb/issues", e)}
                            >
                                <GlobeAltIcon className="w-5 h-5"/>
                                {t('submitIssue')}
                            </Button>
                            <p className="text-sm">{t('communityDescription')}</p>
                        </div>
                        <Separator/>
                        <div className="flex flex-col gap-2">
                            <Label className="text-lg font-semibold">{t('liveChatTitle')}</Label>
                            <Button
                                variant="ghost"
                                className="w-fit gap-sm cursor-not-allowed opacity-60"
                                disabled
                            >
                                <ChatBubbleLeftRightIcon className="w-5 h-5"/>
                                {t('chatButton')}
                            </Button>
                            <p className="text-sm">{t('liveChatDescription')}</p>
                        </div>
                    </div>
                    <div className="flex flex-col items-start gap-sm text-xs text-gray-500 py-4">
                        <div>
                            {t('companyInfo')}
                        </div>
                        <div>
                            {t('urgentNote')} <span
                            className="font-mono bg-gray-100 px-1 rounded">{t('urgentLabel')}</span> {t('urgentNoteEnd')}
                        </div>
                    </div>
                </div>
            </div>
        </InternalPage>
    );
}