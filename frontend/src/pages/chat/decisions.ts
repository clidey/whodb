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

import type {
    AiDecisionAnswer,
    AiDecisionLevel,
    AiDecisionQuestion,
    AiDecisionQuestionInput,
    Column,
    RowsResult,
} from '../../generated/schema-types';
import {AiDecisionAnswerType, AiDecisionQuestionType, SourceMetadataFidelity} from '../../generated/schema-types';

/**
 * One row in a Decisions run. Null in a run's results means the row was never sent; an
 * empty object means it was sent and has no answer yet; otherwise it holds the answers or
 * the error that stopped them.
 */
export type DecisionRowResult = {
    answers?: AiDecisionAnswer[];
    error?: string;
};

/** Translated column headers and cell text for a Decisions results table. */
export type DecisionTableLabels = {
    confidence: (question: string) => string;
    level: (question: string) => string;
    status: string;
    refused: string;
    notSent: string;
    noAnswer: string;
};

/** Serializes one row as the JSON object sent as Decisions input. Repeated column names get a numeric suffix so no value is dropped. */
export function decisionInput(columns: string[], row: string[]): string {
    const record: Record<string, string> = {};
    const seen = new Map<string, number>();
    columns.forEach((column, index) => {
        const count = (seen.get(column) ?? 0) + 1;
        seen.set(column, count);
        record[count === 1 ? column : `${column}_${count}`] = row[index] ?? '';
    });
    return JSON.stringify(record);
}

/** Converts planned questions into RunAIDecision input, keeping only the fields the API takes. */
export function decisionQuestionInputs(questions: AiDecisionQuestion[]): AiDecisionQuestionInput[] {
    return questions.map(question => ({
        Type: question.Type,
        Name: question.Name,
        Instructions: question.Instructions,
        Choices: question.Choices?.map(choice => ({Value: choice.Value, Description: choice.Description ?? undefined})),
        Levels: question.Levels?.map(level => ({Label: level.Label, Description: level.Description ?? undefined})),
    }));
}

/** Returns the label of the level a score is closest to; scores are probability-weighted level indexes. */
export function nearestLevelLabel(score: number, levels: AiDecisionLevel[]): string {
    const index = Math.min(levels.length - 1, Math.max(0, Math.round(score)));
    return levels[index]?.Label ?? '';
}

function formatNumber(value: number | null | undefined): string {
    return value == null ? '' : value.toFixed(2);
}

function answerColumn(name: string): Column {
    return {Name: name, Type: 'TEXT', IsPrimary: false, IsForeignKey: false, MetadataFidelity: SourceMetadataFidelity.Synthetic};
}

/**
 * Builds the results table: each evidence row followed by its answers. Choice questions add a
 * confidence column and score questions add level and confidence columns. A status column is
 * added when a row failed, or when a stopped run left rows unsent or unanswered.
 */
export function decisionResultTable(
    evidence: RowsResult,
    questions: AiDecisionQuestion[],
    results: Array<DecisionRowResult | null>,
    labels: DecisionTableLabels,
    stopped: boolean,
): RowsResult {
    const columns = [...evidence.Columns];
    for (const question of questions) {
        columns.push(answerColumn(question.Name));
        if (question.Type === AiDecisionQuestionType.Score) {
            columns.push(answerColumn(labels.level(question.Name)));
        }
        if (question.Type !== AiDecisionQuestionType.Predicate) {
            columns.push(answerColumn(labels.confidence(question.Name)));
        }
    }
    const status = (result: DecisionRowResult | null): string => {
        if (result?.error != null) return result.error;
        if (!stopped || result?.answers != null) return '';
        return result == null ? labels.notSent : labels.noAnswer;
    };
    const hasStatus = evidence.Rows.some((_, index) => status(results[index] ?? null) !== '');
    if (hasStatus) {
        columns.push(answerColumn(labels.status));
    }

    const rows = evidence.Rows.map((row, index) => {
        const result = results[index] ?? null;
        const cells = [...row];
        for (const question of questions) {
            const answer = result?.answers?.find(candidate => candidate.Name === question.Name);
            const refused = answer?.Type === AiDecisionAnswerType.Refusal;
            switch (question.Type) {
                case AiDecisionQuestionType.Predicate:
                    cells.push(refused ? labels.refused : formatNumber(answer?.Probability));
                    break;
                case AiDecisionQuestionType.Choice:
                    cells.push(refused ? labels.refused : answer?.Choice ?? '', refused ? '' : formatNumber(answer?.Confidence));
                    break;
                case AiDecisionQuestionType.Score:
                    cells.push(
                        refused ? labels.refused : formatNumber(answer?.Score),
                        refused || answer?.Score == null ? '' : nearestLevelLabel(answer.Score, question.Levels ?? []),
                        refused ? '' : formatNumber(answer?.Confidence),
                    );
                    break;
            }
        }
        if (hasStatus) {
            cells.push(status(result));
        }
        return cells;
    });

    return {Columns: columns, Rows: rows, DisableUpdate: true, TotalCount: rows.length};
}
