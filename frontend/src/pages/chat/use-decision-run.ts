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

import {useMutation} from '@apollo/client/react';
import {RunAiDecisionDocument} from '@graphql';
import {useCallback} from 'react';
import {useTranslation} from '@/hooks/use-translation';
import {reduxStorePersistor} from '../../store';
import {HoudiniActions, type IChatMessage} from '../../store/chat';
import {useAppDispatch, useAppSelector} from '../../store/hooks';
import {decisionInput, decisionQuestionInputs} from './decisions';

// Each row is one Decisions request; this many are in flight at once.
const DECISION_CONCURRENCY = 4;

// Active runs by message id. Kept outside React so Stop still works after the card remounts.
const activeRuns = new Map<number, AbortController>();

/**
 * Sends a reviewed decision message's rows, one request per row, to the provider that planned
 * it, and records each row's answers as they arrive. A row is marked sent, and that mark saved,
 * before its request goes out, so even after a reload the chat shows which rows left WhoDB.
 * Nothing is retried.
 */
export function useDecisionRun() {
    const {t} = useTranslation('pages/chat');
    const dispatch = useAppDispatch();
    const [runAIDecision] = useMutation(RunAiDecisionDocument);
    const providers = useAppSelector(state => state.aiModels.modelTypes);

    const run = useCallback(async (sessionId: string, message: IChatMessage) => {
        const {id, Decision: decision, Result: result, decisionProvider} = message;
        const provider = providers.find(candidate => candidate.id === decisionProvider?.id);
        if (id == null || decision == null || result == null || provider == null || activeRuns.has(id)) {
            return;
        }

        const controller = new AbortController();
        activeRuns.set(id, controller);
        dispatch(HoudiniActions.startDecisionRun({sessionId, id, rowCount: result.Rows.length}));
        const columns = result.Columns.map(column => column.Name);
        const questions = decisionQuestionInputs(decision.Questions);
        let next = 0;
        const sendRows = async () => {
            while (!controller.signal.aborted && next < result.Rows.length) {
                const index = next++;
                dispatch(HoudiniActions.setDecisionRowResult({sessionId, id, index, result: {}}));
                await reduxStorePersistor.flush();
                if (controller.signal.aborted) {
                    // Stopped before this row's request went out, so it was never sent.
                    dispatch(HoudiniActions.setDecisionRowResult({sessionId, id, index, result: null}));
                    return;
                }
                try {
                    const {data} = await runAIDecision({
                        variables: {
                            input: {
                                ModelType: provider.modelType,
                                ProviderId: provider.id,
                                Token: provider.token,
                                Model: decision.Model,
                                Input: decisionInput(columns, result.Rows[index]),
                                Questions: questions,
                            },
                        },
                        context: {fetchOptions: {signal: controller.signal}},
                    });
                    if (controller.signal.aborted) return;
                    dispatch(HoudiniActions.setDecisionRowResult({
                        sessionId,
                        id,
                        index,
                        result: data ? {answers: data.RunAIDecision.Answers} : {error: t('decisionRowFailed')},
                    }));
                } catch (error) {
                    if (controller.signal.aborted) return;
                    dispatch(HoudiniActions.setDecisionRowResult({
                        sessionId,
                        id,
                        index,
                        result: {error: error instanceof Error && error.message ? error.message : t('decisionRowFailed')},
                    }));
                }
            }
        };
        await Promise.all(Array.from({length: Math.min(DECISION_CONCURRENCY, result.Rows.length)}, sendRows));
        activeRuns.delete(id);
        dispatch(HoudiniActions.finishDecisionRun({sessionId, id, status: controller.signal.aborted ? 'stopped' : 'complete'}));
        await reduxStorePersistor.flush();
    }, [dispatch, providers, runAIDecision, t]);

    const stop = useCallback((id: number) => {
        activeRuns.get(id)?.abort();
    }, []);

    return {run, stop};
}
