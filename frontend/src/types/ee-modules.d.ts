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

declare module '@ee/pages/raw-execute/index' {
  export const plugins: any;
  export const ActionOptions: any;
  export const ActionOptionIcons: any;
  export default { plugins, ActionOptions, ActionOptionIcons };
}