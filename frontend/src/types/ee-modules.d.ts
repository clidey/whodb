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

// Type declarations for EE modules in CE builds
// This allows any import from @ee/* to be typed as 'any'

declare module '@ee/*' {
  const content: any;
  export = content;
}

declare module '@ee/icons' {
  export const EEIcons: any;
  export default EEIcons;
}

declare module '@ee/config.tsx' {
  export const eeDatabaseTypes: any;
  export const eeFeatures: any;
  export const eeExtensions: Record<string, any>;
  export const eeSources: Record<string, any>;
  export const isEEDatabase: any;
  export const isEENoSQLDatabase: any;
  export const getEEDatabaseStorageUnitLabel: any;
  export default eeDatabaseTypes;
}

declare module '@ee/index' {
  export const isEENoSQLDatabase: any;
  export const getEEDatabaseStorageUnitLabel: any;
  export const AnalyzeGraph: any;
  export default null;
}

declare module '@ee/components/charts/line-chart' {
  export const LineChart: any;
  export default LineChart;
}

declare module '@ee/components/charts/pie-chart' {
  export const PieChart: any;
  export default PieChart;
}

declare module '@ee/components/export' {
  export const Export: any;
  export default Export;
}

declare module '@ee/pages/raw-execute/index' {
  export const plugins: any;
  export const ActionOptions: any;
  export const ActionOptionIcons: any;
  export default { plugins, ActionOptions, ActionOptionIcons };
}