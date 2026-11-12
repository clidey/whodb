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

import type {AiDecisionQuestion, RowsResult} from '../../generated/schema-types';
import {AiDecisionAnswerType, AiDecisionQuestionType, SourceMetadataFidelity} from '../../generated/schema-types';
import {decisionInput, decisionQuestionInputs, decisionResultTable, nearestLevelLabel, type DecisionTableLabels} from './decisions';

// Compares JSON, which is also what reaches the API, so undefined fields count as absent.
const assertEqual = (actual: unknown, expected: unknown, message: string) => {
    const got = JSON.stringify(actual);
    const want = JSON.stringify(expected);
    if (got !== want) {
        throw new Error(`${message}: expected ${want}, got ${got}`);
    }
};

const labels: DecisionTableLabels = {
    confidence: name => `${name} confidence`,
    level: name => `${name} level`,
    status: 'Decision status',
    refused: 'Refused',
    notSent: 'Not sent',
    noAnswer: 'Stopped before an answer',
};

const questions: AiDecisionQuestion[] = [
    {Type: AiDecisionQuestionType.Choice, Name: 'team', Instructions: 'Which team?', Choices: [{Value: 'billing', Description: 'Payments.'}, {Value: 'other', Description: null}]},
    {Type: AiDecisionQuestionType.Predicate, Name: 'refund_requested', Instructions: 'The customer asks for a refund.'},
    {Type: AiDecisionQuestionType.Score, Name: 'severity', Instructions: 'How severe?', Levels: [{Label: 'Minor'}, {Label: 'Major'}, {Label: 'Blocking'}]},
];

const evidence: RowsResult = {
    Columns: [
        {Name: 'id', Type: 'INTEGER', IsPrimary: true, IsForeignKey: false, MetadataFidelity: SourceMetadataFidelity.Exact},
        {Name: 'subject', Type: 'TEXT', IsPrimary: false, IsForeignKey: false, MetadataFidelity: SourceMetadataFidelity.Exact},
    ],
    Rows: [['1', 'Charged twice'], ['2', 'Export fails'], ['3', 'Spam'], ['4', 'Login loop']],
    DisableUpdate: false,
    TotalCount: 4,
};

// Rows become JSON objects keyed by column, and repeated names keep every value.
assertEqual(decisionInput(['id', 'subject'], ['7', 'Charged "twice"']), '{"id":"7","subject":"Charged \\"twice\\""}', 'row input');
assertEqual(decisionInput(['id', 'id', 'id', 'note'], ['1', '2', '3', '']), '{"id":"1","id_2":"2","id_3":"3","note":""}', 'repeated columns');

// Only the fields RunAIDecision takes are kept, and null descriptions are dropped.
assertEqual(decisionQuestionInputs([{...questions[0], __typename: 'AIDecisionQuestion'}, questions[1]]), [
    {Type: AiDecisionQuestionType.Choice, Name: 'team', Instructions: 'Which team?', Choices: [{Value: 'billing', Description: 'Payments.'}, {Value: 'other', Description: undefined}], Levels: undefined},
    {Type: AiDecisionQuestionType.Predicate, Name: 'refund_requested', Instructions: 'The customer asks for a refund.', Choices: undefined, Levels: undefined},
], 'question inputs');

// Scores are probability-weighted level indexes, so they round to the closest level.
const levels = [{Label: 'Minor'}, {Label: 'Major'}, {Label: 'Blocking'}];
assertEqual(nearestLevelLabel(0.4, levels), 'Minor', 'nearest level for 0.4');
assertEqual(nearestLevelLabel(1.5, levels), 'Blocking', 'nearest level for 1.5');
assertEqual(nearestLevelLabel(7, levels), 'Blocking', 'nearest level for 7');
assertEqual(nearestLevelLabel(-1, levels), 'Minor', 'nearest level for -1');

// A finished run adds answer columns, shows refusals, and adds a status column only for failures.
const finished = decisionResultTable(evidence, questions, [
    {answers: [
        {Type: AiDecisionAnswerType.Choice, Name: 'team', Choice: 'billing', Confidence: 0.934},
        {Type: AiDecisionAnswerType.Predicate, Name: 'refund_requested', Probability: 0.97},
        {Type: AiDecisionAnswerType.Score, Name: 'severity', Score: 1.1, Confidence: 0.55},
    ]},
    {answers: [
        {Type: AiDecisionAnswerType.Refusal, Name: 'team'},
        {Type: AiDecisionAnswerType.Predicate, Name: 'refund_requested', Probability: 0.02},
        {Type: AiDecisionAnswerType.Refusal, Name: 'severity'},
    ]},
    {error: 'failed to create decision: invalid input'},
    {answers: [
        {Type: AiDecisionAnswerType.Choice, Name: 'team', Choice: 'other', Confidence: 0.5},
        {Type: AiDecisionAnswerType.Predicate, Name: 'refund_requested', Probability: 0},
        {Type: AiDecisionAnswerType.Score, Name: 'severity', Score: 2, Confidence: 1},
    ]},
], labels, false);
assertEqual(finished.Columns.map(column => column.Name), [
    'id', 'subject', 'team', 'team confidence', 'refund_requested', 'severity', 'severity level', 'severity confidence', 'Decision status',
], 'result columns');
assertEqual(finished.Columns[2].MetadataFidelity, SourceMetadataFidelity.Synthetic, 'answer column fidelity');
assertEqual(finished.Rows, [
    ['1', 'Charged twice', 'billing', '0.93', '0.97', '1.10', 'Major', '0.55', ''],
    ['2', 'Export fails', 'Refused', '', '0.02', 'Refused', '', '', ''],
    ['3', 'Spam', '', '', '', '', '', '', 'failed to create decision: invalid input'],
    ['4', 'Login loop', 'other', '0.50', '0.00', '2.00', 'Blocking', '1.00', ''],
], 'result rows');
assertEqual(finished.TotalCount, 4, 'result count');

// While a run is going, unanswered rows stay blank and no status column appears.
const running = decisionResultTable(evidence, questions, [{}, null, null, null], labels, false);
assertEqual(running.Columns.length, 8, 'running columns');
assertEqual(running.Rows[0], ['1', 'Charged twice', '', '', '', '', '', ''], 'running row');

// A stopped run says which rows were sent without an answer and which never left WhoDB.
const stopped = decisionResultTable(evidence, questions, [
    {answers: [{Type: AiDecisionAnswerType.Predicate, Name: 'refund_requested', Probability: 0.4}]},
    {},
    null,
    null,
], labels, true);
assertEqual(stopped.Rows.map(row => row[row.length - 1]), ['', 'Stopped before an answer', 'Not sent', 'Not sent'], 'stopped statuses');
