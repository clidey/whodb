/**
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

import { ComponentType } from 'react';

// Define proper types for EE components
export interface AnalyzeGraphProps {
    graph: any; // Replace with actual graph type
}

export interface LineChartProps {
    data: any[];
    xKey: string;
    yKey: string;
    width?: number;
    height?: number;
}

export interface PieChartProps {
    data: any[];
    dataKey: string;
    nameKey: string;
    width?: number;
    height?: number;
}

export interface ThemeConfigProps {
    children?: React.ReactNode;
}

// Type-safe EE component definitions
export type EEComponentTypes = {
    AnalyzeGraph: ComponentType<AnalyzeGraphProps> | null;
    LineChart: ComponentType<LineChartProps> | null;
    PieChart: ComponentType<PieChartProps> | null;
    ThemeConfig: ComponentType<ThemeConfigProps> | null;
};