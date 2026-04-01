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

/**
 * Enterprise Edition Type Definitions
 * 
 * These types define the interfaces for EE features but don't expose any implementation.
 * The actual implementations remain in the EE module.
 */

import {ComponentType} from 'react';
import {TypeDefinition, SSLModeOption} from './database-types';

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
    text: string;
}

export interface PieChartProps {
    data: string[][];
    columns: string[];
    text: string;
}

export interface BarChartProps {
    data: string[][];
    columns: string[];
    text: string;
}

export interface AreaChartProps {
    data: string[][];
    columns: string[];
    text: string;
}

export interface ScatterChartProps {
    data: string[][];
    columns: string[];
    text: string;
}

export interface RadarChartProps {
    data: string[][];
    columns: string[];
    text: string;
}

export interface TreemapChartProps { data: string[][]; columns: string[]; text: string; }
export interface FunnelChartProps { data: string[][]; columns: string[]; text: string; }
export interface BubbleChartProps { data: string[][]; columns: string[]; text: string; }
export interface WaterfallChartProps { data: string[][]; columns: string[]; text: string; }
export interface HeatmapChartProps { data: string[][]; columns: string[]; text: string; }
export interface GeoMapChartProps { data: string[][]; columns: string[]; text: string; }

// Type-safe EE component registry
export type EEComponentTypes = {
    AnalyzeGraph: ComponentType<AnalyzeGraphProps> | null;
    LineChart: ComponentType<LineChartProps> | null;
    PieChart: ComponentType<PieChartProps> | null;
    BarChart: ComponentType<BarChartProps> | null;
    AreaChart: ComponentType<AreaChartProps> | null;
    ScatterChart: ComponentType<ScatterChartProps> | null;
    RadarChart: ComponentType<RadarChartProps> | null;
    TreemapChart: ComponentType<TreemapChartProps> | null;
    FunnelChart: ComponentType<FunnelChartProps> | null;
    BubbleChart: ComponentType<BubbleChartProps> | null;
    WaterfallChart: ComponentType<WaterfallChartProps> | null;
    HeatmapChart: ComponentType<HeatmapChartProps> | null;
    GeoMapChart: ComponentType<GeoMapChartProps> | null;
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
    sampleDatabaseTour: boolean; // Show sample database tour on login page
    autoStartTourOnLogin: boolean; // Auto-start tour when logging in with sample database
    sqlAgent: boolean; // SQL Agent with planning, thinking, and tool use
}

// Settings defaults that can be overridden by EE
export interface SettingsDefaults {
    whereConditionMode?: 'popover' | 'sheet';
    disableAnimations?: boolean;
}

// EE Database type definition
export interface EEDatabaseType {
    id: string;
    label: string;
    iconName: string;
    extra: Record<string, string>;
    fields?: {
        hostname?: boolean;
        username?: boolean;
        password?: boolean;
        database?: boolean;
    };
    operators?: string[];
    typeDefinitions?: TypeDefinition[];
    aliasMap?: Record<string, string>;
    supportsModifiers?: boolean;
    supportsScratchpad?: boolean;
    supportsSchema?: boolean;
    supportsDatabaseSwitching?: boolean;
    usesSchemaForGraph?: boolean;
    sslModes?: SSLModeOption[];
}