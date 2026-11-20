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

import {Button} from "@clidey/ux";
import type {RowsResult} from '@graphql';
import {AiDecisionQuestionType} from '@graphql';
import type {FC, ReactNode} from "react";
import {useId, useMemo, useState} from "react";
import {useTranslation} from '@/hooks/use-translation';
import {Spinner} from "../../components/loading";
import {WhoDBChatIcon} from "../../components/whodb-chat-icon";
import {ANALYTICS_EVENTS} from "../../config/analytics-events";
import {countBucket, trackFrontendIntent} from "../../config/frontend-analytics";
import {reduxStorePersistor} from "../../store";
import {HoudiniActions, type IChatMessage} from "../../store/chat";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {decisionResultTable} from "./decisions";
import {useDecisionRun} from "./use-decision-run";

type DecisionCardProps = {
    message: IChatMessage;
    sessionId: string;
    activity?: ReactNode;
    renderTable: (data: RowsResult, complete: boolean) => ReactNode;
};

/**
 * Shows a planned Decisions run. Until the user sends it, the card lists the questions, where
 * the rows go and the exact rows; nothing leaves WhoDB before "Send". Afterwards it shows
 * progress and each row's answers.
 */
export const DecisionCard: FC<DecisionCardProps> = ({message, sessionId, activity, renderTable}) => {
    const {t} = useTranslation('pages/chat');
    const dispatch = useAppDispatch();
    const providers = useAppSelector(state => state.aiModels.modelTypes);
    const {run, stop} = useDecisionRun();
    const [showDescriptions, setShowDescriptions] = useState(false);
    const questionsId = useId();
    const decision = message.Decision;
    const evidence = message.Result;
    const decisionRun = message.decisionRun;
    const status = decisionRun?.status;
    const results = decisionRun?.results ?? [];
    const rowCount = evidence?.Rows.length ?? 0;
    const sent = results.filter(result => result != null).length;
    const answered = results.filter(result => result?.answers != null || result?.error != null).length;
    const providerAvailable = providers.some(provider => provider.id === message.decisionProvider?.id);
    const hasDescriptions = decision?.Questions.some(question =>
        [...(question.Choices ?? []), ...(question.Levels ?? [])].some(option => Boolean(option.Description))) ?? false;

    const table = useMemo(() => {
        if (decision == null || evidence == null || decisionRun == null || decisionRun.status === 'declined') {
            return evidence;
        }
        return decisionResultTable(evidence, decision.Questions, decisionRun.results, {
            confidence: name => t('decisionConfidenceColumn', {name}),
            level: name => t('decisionLevelColumn', {name}),
            status: t('decisionStatusColumn'),
            refused: t('decisionRefused'),
            notSent: t('decisionNotSent'),
            noAnswer: t('decisionNoAnswer'),
        }, decisionRun.status === 'stopped');
    }, [decision, evidence, decisionRun, t]);

    if (decision == null || evidence == null || message.id == null) {
        return null;
    }
    const id = message.id;
    const reviewing = status == null && message.RequiresConfirmation;
    const providerName = message.decisionProvider?.name ?? '';
    const analytics = {row_count_bucket: countBucket(rowCount), question_count_bucket: countBucket(decision.Questions.length)};
    // Category values and levels stay visible; their descriptions, also sent with each row, are one click away.
    const optionsClass = showDescriptions ? 'ce-chat-decision-options' : 'ce-chat-decision-options ce-chat-decision-options-compact';
    const questionTypes = {
        [AiDecisionQuestionType.Predicate]: t('decisionTypePredicate'),
        [AiDecisionQuestionType.Choice]: t('decisionTypeChoice'),
        [AiDecisionQuestionType.Score]: t('decisionTypeScore'),
    };

    const heading = status === 'running'
        ? {title: t('decisionsRunningTitle'), detail: t('decisionsProgress', {answered, total: rowCount})}
        : status === 'complete'
            ? {title: t('decisionsTitle'), detail: t('decisionsAnswered', {count: rowCount})}
            : status === 'stopped'
                ? {title: t('decisionsStoppedTitle'), detail: t('decisionsStopped', {sent, total: rowCount})}
                : status === 'declined'
                    ? {title: t('decisionsDeclinedTitle'), detail: t('decisionsNothingSent')}
                    : reviewing
                        ? {title: t('reviewDecisions'), detail: t('decisionsNeedsConsent')}
                        : {title: t('decisionsNoRowsTitle'), detail: t('decisionsNothingToSend')};

    const handleSend = () => {
        trackFrontendIntent(ANALYTICS_EVENTS.CHAT_DECISION_RUN_CONFIRMED, analytics);
        void run(sessionId, message);
    };
    const handleDecline = () => {
        trackFrontendIntent(ANALYTICS_EVENTS.CHAT_DECISION_RUN_DECLINED, analytics);
        dispatch(HoudiniActions.declineDecision({sessionId, id}));
        void reduxStorePersistor.flush();
    };
    const handleStop = () => {
        trackFrontendIntent(ANALYTICS_EVENTS.CHAT_DECISION_RUN_STOPPED, analytics);
        stop(id);
    };

    return <div className="ce-chat-change-card ce-chat-decision-card flex flex-col w-[calc(100%-50px)] max-w-full min-w-0" data-testid="decision-message" data-decision-status={status ?? (reviewing ? 'review' : 'empty')}>
        {activity}
        <div className="ce-chat-change-heading">
            <WhoDBChatIcon name="list" />
            <div>
                <strong>{heading.title}</strong>
                <span role={status === 'running' ? 'status' : undefined}>{heading.detail}</span>
            </div>
            {status === 'running' && <Spinner />}
        </div>
        <div className="ce-chat-decision-body">
            {reviewing && <>
                <p>{t('decisionsConsentNotice', {provider: providerName})}</p>
                <ul className="ce-chat-decision-facts" data-testid="decision-facts">
                    <li>{t('decisionsDestination', {provider: providerName, model: decision.Model})}</li>
                    <li>{t('decisionsColumns', {columns: evidence.Columns.map(column => column.Name).join(', ')})}</li>
                    <li>{t('decisionsRowCount', {count: rowCount})}</li>
                    {decision.Truncated && <li>{t('decisionsTruncated', {count: rowCount})}</li>}
                </ul>
            </>}
            {status == null && !reviewing && <p>{t('decisionsNoRows')}</p>}
            {status !== 'declined' && <>
                <ol className="ce-chat-decision-questions" id={questionsId} aria-label={t('decisionsQuestions')}>
                    {decision.Questions.map(question => <li key={question.Name}>
                        <div className="ce-chat-decision-question-heading"><code>{question.Name}</code><b>{questionTypes[question.Type]}</b></div>
                        <p>{question.Instructions}</p>
                        {question.Choices != null && question.Choices.length > 0 && <ul className={optionsClass}>
                            {question.Choices.map(choice => <li key={choice.Value}><code>{choice.Value}</code>{showDescriptions && choice.Description && <span>{choice.Description}</span>}</li>)}
                        </ul>}
                        {question.Levels != null && question.Levels.length > 0 && <ol className={optionsClass} start={0}>
                            {question.Levels.map(level => <li key={level.Label}><code>{level.Label}</code>{showDescriptions && level.Description && <span>{level.Description}</span>}</li>)}
                        </ol>}
                    </li>)}
                </ol>
                {hasDescriptions && <button type="button" className="ce-chat-decision-descriptions-toggle" aria-expanded={showDescriptions} aria-controls={questionsId} onClick={() => { setShowDescriptions(shown => !shown); }} data-testid="decision-descriptions-toggle">
                    {showDescriptions ? t('decisionsHideDescriptions') : t('decisionsShowDescriptions')}
                </button>}
            </>}
            <details className="ce-chat-decision-query">
                <summary>{t('decisionsQuery')}</summary>
                <pre className="ce-chat-result-sql">{decision.Query}</pre>
            </details>
            {status !== 'declined' && rowCount > 0 && table != null && renderTable(table, status === 'complete')}
            {(status === 'complete' || status === 'stopped') && <p className="ce-chat-decision-note">{t('decisionsEstimateNote')}</p>}
            {reviewing && !providerAvailable && <p className="ce-chat-decision-note" role="alert">{t('decisionsProviderMissing')}</p>}
        </div>
        {reviewing && <div className="ce-chat-decision-actions">
            <Button variant="outline" size="sm" onClick={handleDecline} data-testid="decision-decline">{t('decisionsDontSend')}</Button>
            <Button size="sm" onClick={handleSend} disabled={!providerAvailable} data-testid="decision-send">{t('decisionsSend', {count: rowCount})}</Button>
        </div>}
        {status === 'running' && <div className="ce-chat-decision-actions">
            <Button variant="outline" size="sm" onClick={handleStop} data-testid="decision-stop">{t('decisionsStop')}</Button>
        </div>}
    </div>;
};
