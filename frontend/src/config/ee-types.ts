/**
 * Enterprise Edition Type Definitions
 * 
 * These types define the interfaces for EE features but don't expose any implementation.
 * The actual implementations remain in the EE module.
 */

import { ComponentType } from 'react';

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

// ThemeConfig is not a component but a configuration object
export interface ThemeConfigType {
    name?: string;
    logo: string;
    layout: {
        background: string;
        sidebar: string;
        sidebarItem: string;
        chat: {
            background: string;
            user: string;
            houdini: string;
        };
        graph: string;
    };
    components: {
        card: string;
        text: string;
        brandText: string;
        icon: string;
        input: string;
        button: string;
        actionButton: string;
        dropdown: string;
        dropdownPanel: string;
        toggle: string;
        graphCard: string;
        breadcrumb: string;
        table: {
            header: string;
            evenRow: string;
            oddRow: string;
        };
    };
}

// Type-safe EE component registry
export type EEComponentTypes = {
    AnalyzeGraph: ComponentType<AnalyzeGraphProps> | null;
    LineChart: ComponentType<LineChartProps> | null;
    PieChart: ComponentType<PieChartProps> | null;
    ThemeConfig: ThemeConfigType | null;
};

// Feature flags for Enterprise Edition
export interface FeatureFlags {
    analyzeView: boolean;
    customTheme: boolean;
    dataVisualization: boolean; // For charts (line, pie)
    aiChat: boolean; // For Houdini AI assistant
    multiProfile: boolean; // For saving multiple connection profiles
    advancedDatabases: boolean; // For MSSQL, Oracle, DynamoDB
    contactUsPage: boolean; // Show Contact Us page (disabled in EE)
    settingsPage: boolean; // Show Settings page (disabled in EE)
}

// EE Database type definition
export interface EEDatabaseType {
    id: string;
    label: string;
    iconName: string; // Name of icon to resolve from Icons.Logos
    extra: Record<string, string>;
}