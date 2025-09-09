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

/**
 * Enterprise Edition Type Definitions
 * 
 * These types define the interfaces for EE features but don't expose any implementation.
 * The actual implementations remain in the EE module.
 */

import {ComponentType} from 'react';

// Types from analyze-view component
type IPlanNode = {
    "Node Type": string;
    "Hash Cond"?: string;
    "Join Type"?: string;
    "Relation Name"?: string;
    "Actual Rows"?: number;
    "Actual Total Time"?: number;
    Plans?: IPlanNode[];
}

type IExplainAnalyzeResult = {
    Plan: IPlanNode;
    "Execution Time": number;
}

// Component prop types
export interface AnalyzeGraphProps {
    data: IExplainAnalyzeResult;
}

export interface LineChartProps {
    data: string[][];
    columns: string[];
}

export interface PieChartProps {
    data: string[][];
    columns: string[];
}

// Type-safe EE component registry
export type EEComponentTypes = {
    AnalyzeGraph: ComponentType<AnalyzeGraphProps> | null;
    LineChart: ComponentType<LineChartProps> | null;
    PieChart: ComponentType<PieChartProps> | null;
};

// Feature flags for Enterprise Edition
export interface FeatureFlags {
    analyzeView: boolean;
    explainView: boolean;
    generateView: boolean;
    customTheme: boolean;
    dataVisualization: boolean; // For charts (line, pie)
    aiChat: boolean; // For Houdini AI assistant
    multiProfile: boolean; // For saving multiple connection profiles
    advancedDatabases: boolean; // For additional enterprise databases
    contactUsPage: boolean; // Show Contact Us page (disabled in EE)
    settingsPage: boolean; // Show Settings page (disabled in EE)
}

// EE Database type definition
export interface EEDatabaseType {
    id: string;
    label: string;
    iconName: string; // Name of icon to resolve from Icons.Logos
    extra: Record<string, string>;
    fields?: {
        hostname?: boolean;
        username?: boolean;
        password?: boolean;
        database?: boolean;
    };
    operators?: string[]; // Valid SQL operators for this database type
    dataTypes?: string[]; // Valid data types for creating tables/collections
    supportsModifiers?: boolean; // Whether this database supports field modifiers (primary, nullable)
    supportsScratchpad?: boolean; // Whether this database supports scratchpad/raw query execution
    supportsSchema?: boolean; // Whether this database supports schemas
    supportsDatabaseSwitching?: boolean; // Whether this database supports switching between databases in the UI
    usesSchemaForGraph?: boolean; // Whether this database should use the schema field (true) or database field (false) for graph queries
}