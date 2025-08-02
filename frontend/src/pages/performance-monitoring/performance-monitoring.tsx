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

import { FC, useCallback, useEffect, useState } from "react";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { useAppSelector } from "../../store/hooks";
import { Line, Bar, Doughnut } from 'react-chartjs-2';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    BarElement,
    ArcElement,
    Title,
    Tooltip,
    Legend,
    ChartOptions,
} from 'chart.js';
import { SelectInput, Text } from "../../components/input";
import { usePerformanceMetricsQuery } from "./performance-metrics.generated";
import { Navigate } from "react-router-dom";

// Register ChartJS components
ChartJS.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    BarElement,
    ArcElement,
    Title,
    Tooltip,
    Legend
);

const TIME_RANGES = [
    { label: "Last 5 minutes", value: "5m" },
    { label: "Last 15 minutes", value: "15m" },
    { label: "Last 30 minutes", value: "30m" },
    { label: "Last 1 hour", value: "1h" },
    { label: "Last 6 hours", value: "6h" },
    { label: "Last 24 hours", value: "24h" },
    { label: "Last 7 days", value: "7d" },
];

const METRIC_TYPES = {
    query_latency: "Query Latency",
    query_count: "Query Count",
    connection_count: "Connection Count",
    error_count: "Error Count",
    cpu_usage: "CPU Usage",
    memory_usage: "Memory Usage",
    disk_io: "Disk I/O",
    cache_hit_ratio: "Cache Hit Ratio",
    transaction_count: "Transaction Count",
    lock_wait_time: "Lock Wait Time",
};

export const PerformanceMonitoringPage: FC = () => {
    const performanceMonitoringEnabled = useAppSelector(state => state.settings.performanceMonitoringEnabled);
    const performanceMetricsConfig = useAppSelector(state => state.settings.performanceMetricsConfig);
    const selectedDatabase = useAppSelector(state => state.database.databaseName);
    
    const [timeRange, setTimeRange] = useState("1h");
    const [selectedMetric, setSelectedMetric] = useState("query_latency");
    
    // Calculate time range
    const now = new Date();
    const startTime = new Date();
    switch (timeRange) {
        case "5m": startTime.setMinutes(now.getMinutes() - 5); break;
        case "15m": startTime.setMinutes(now.getMinutes() - 15); break;
        case "30m": startTime.setMinutes(now.getMinutes() - 30); break;
        case "1h": startTime.setHours(now.getHours() - 1); break;
        case "6h": startTime.setHours(now.getHours() - 6); break;
        case "24h": startTime.setHours(now.getHours() - 24); break;
        case "7d": startTime.setDate(now.getDate() - 7); break;
    }
    
    // Get enabled metrics
    const enabledMetrics = Object.entries(performanceMetricsConfig)
        .filter(([_, enabled]) => enabled)
        .map(([metric, _]) => metric);
    
    const { data, loading, error, refetch } = usePerformanceMetricsQuery({
        variables: {
            query: {
                startTime: startTime.toISOString(),
                endTime: now.toISOString(),
                metricTypes: enabledMetrics,
                database: selectedDatabase || undefined,
            }
        },
        pollInterval: 30000, // Refresh every 30 seconds
        skip: !performanceMonitoringEnabled,
    });
    
    const handleTimeRangeChange = useCallback((value: string) => {
        setTimeRange(value);
    }, []);
    
    const handleMetricChange = useCallback((value: string) => {
        setSelectedMetric(value);
    }, []);
    
    // Auto-refresh
    useEffect(() => {
        if (performanceMonitoringEnabled) {
            const interval = setInterval(() => {
                refetch();
            }, 30000);
            return () => clearInterval(interval);
        }
    }, [performanceMonitoringEnabled, refetch]);
    
    if (!performanceMonitoringEnabled) {
        return <Navigate to={InternalRoutes.Settings!.path} replace />;
    }
    
    // Process data for charts
    const selectedMetricData = data?.PerformanceMetrics?.find(m => m.metricType === selectedMetric);
    
    const lineChartData = {
        labels: selectedMetricData?.values.map(v => new Date(v.timestamp).toLocaleTimeString()) || [],
        datasets: [{
            label: METRIC_TYPES[selectedMetric as keyof typeof METRIC_TYPES],
            data: selectedMetricData?.values.map(v => v.value) || [],
            borderColor: 'rgb(75, 192, 192)',
            backgroundColor: 'rgba(75, 192, 192, 0.2)',
            tension: 0.1,
        }]
    };
    
    const lineChartOptions: ChartOptions<'line'> = {
        responsive: true,
        plugins: {
            legend: {
                position: 'top' as const,
            },
            title: {
                display: true,
                text: `${METRIC_TYPES[selectedMetric as keyof typeof METRIC_TYPES]} Over Time`,
            },
        },
        scales: {
            y: {
                beginAtZero: true,
            }
        }
    };
    
    // Query distribution chart
    const queryTypeData = data?.PerformanceMetrics?.find(m => m.metricType === 'query_count');
    const queryTypes = ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'OTHER'];
    const queryTypeCounts = queryTypes.map(type => {
        const values = queryTypeData?.values.filter(v => 
            v.labels?.find(l => l.Key === 'query_type' && l.Value === type)
        );
        return values?.reduce((sum, v) => sum + v.value, 0) || 0;
    });
    
    const barChartData = {
        labels: queryTypes,
        datasets: [{
            label: 'Query Count by Type',
            data: queryTypeCounts,
            backgroundColor: [
                'rgba(75, 192, 192, 0.6)',
                'rgba(54, 162, 235, 0.6)',
                'rgba(255, 206, 86, 0.6)',
                'rgba(255, 99, 132, 0.6)',
                'rgba(153, 102, 255, 0.6)',
            ],
        }]
    };
    
    // Error rate doughnut chart
    const errorData = data?.PerformanceMetrics?.find(m => m.metricType === 'error_count');
    const totalQueries = queryTypeData?.values.reduce((sum, v) => sum + v.value, 0) || 0;
    const totalErrors = errorData?.values.reduce((sum, v) => sum + v.value, 0) || 0;
    
    const doughnutData = {
        labels: ['Successful', 'Failed'],
        datasets: [{
            data: [totalQueries - totalErrors, totalErrors],
            backgroundColor: [
                'rgba(75, 192, 192, 0.6)',
                'rgba(255, 99, 132, 0.6)',
            ],
        }]
    };
    
    return (
        <InternalPage routes={[InternalRoutes.PerformanceMonitoring!]}>
            <div className="flex flex-col gap-6">
                <div className="flex justify-between items-center">
                    <h1 className="text-2xl font-semibold text-neutral-700 dark:text-neutral-300">
                        Database Performance Monitoring
                    </h1>
                    <div className="flex gap-4">
                        <SelectInput
                            value={timeRange}
                            onChange={handleTimeRangeChange}
                            items={TIME_RANGES}
                            placeholder="Select time range"
                        />
                        <SelectInput
                            value={selectedMetric}
                            onChange={handleMetricChange}
                            items={enabledMetrics.map(m => ({
                                label: METRIC_TYPES[m as keyof typeof METRIC_TYPES],
                                value: m
                            }))}
                            placeholder="Select metric"
                        />
                    </div>
                </div>
                
                {loading && (
                    <div className="flex justify-center items-center h-64">
                        <Text label="Loading metrics..." />
                    </div>
                )}
                
                {error && (
                    <div className="bg-red-100 dark:bg-red-900 p-4 rounded">
                        <Text label={`Error loading metrics: ${error.message}`} />
                    </div>
                )}
                
                {!loading && !error && data && (
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                        {/* Main metric chart */}
                        <div className="bg-white dark:bg-neutral-800 p-6 rounded-lg shadow">
                            <Line data={lineChartData} options={lineChartOptions} />
                        </div>
                        
                        {/* Query distribution */}
                        {performanceMetricsConfig.query_count && (
                            <div className="bg-white dark:bg-neutral-800 p-6 rounded-lg shadow">
                                <h3 className="text-lg font-semibold mb-4 text-neutral-700 dark:text-neutral-300">
                                    Query Distribution
                                </h3>
                                <Bar data={barChartData} />
                            </div>
                        )}
                        
                        {/* Error rate */}
                        {performanceMetricsConfig.error_count && (
                            <div className="bg-white dark:bg-neutral-800 p-6 rounded-lg shadow">
                                <h3 className="text-lg font-semibold mb-4 text-neutral-700 dark:text-neutral-300">
                                    Success vs Error Rate
                                </h3>
                                <Doughnut data={doughnutData} />
                            </div>
                        )}
                        
                        {/* Summary stats */}
                        <div className="bg-white dark:bg-neutral-800 p-6 rounded-lg shadow">
                            <h3 className="text-lg font-semibold mb-4 text-neutral-700 dark:text-neutral-300">
                                Summary Statistics
                            </h3>
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <Text label="Total Queries" className="text-sm text-neutral-600 dark:text-neutral-400" />
                                    <Text label={totalQueries.toLocaleString()} className="text-2xl font-bold" />
                                </div>
                                <div>
                                    <Text label="Total Errors" className="text-sm text-neutral-600 dark:text-neutral-400" />
                                    <Text label={totalErrors.toLocaleString()} className="text-2xl font-bold text-red-500" />
                                </div>
                                {selectedMetricData && selectedMetricData.values.length > 0 && (
                                    <>
                                        <div>
                                            <Text label="Average" className="text-sm text-neutral-600 dark:text-neutral-400" />
                                            <Text 
                                                label={`${(selectedMetricData.values.reduce((sum, v) => sum + v.value, 0) / selectedMetricData.values.length).toFixed(2)}`} 
                                                className="text-2xl font-bold" 
                                            />
                                        </div>
                                        <div>
                                            <Text label="Peak" className="text-sm text-neutral-600 dark:text-neutral-400" />
                                            <Text 
                                                label={`${Math.max(...selectedMetricData.values.map(v => v.value)).toFixed(2)}`} 
                                                className="text-2xl font-bold" 
                                            />
                                        </div>
                                    </>
                                )}
                            </div>
                        </div>
                    </div>
                )}
                
                {!loading && !error && (!data || data.PerformanceMetrics?.length === 0) && (
                    <div className="bg-yellow-100 dark:bg-yellow-900 p-6 rounded-lg">
                        <Text label="No performance data available yet. Make sure performance monitoring is enabled and you have executed some queries." />
                    </div>
                )}
            </div>
        </InternalPage>
    );
};