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

import React from 'react';
import { Card } from './card';
import { Icons } from './icons';

export const EEFeatureCard: React.FC<{ feature: string; description?: string }> = ({ feature, description }) => {
    return (
        <Card className="p-8 text-center">
            <div className="flex flex-col items-center space-y-4">
                {Icons.Star}
                <h3 className="text-xl font-semibold">{feature}</h3>
                <p className="text-gray-600 dark:text-gray-400">
                    {description || 'This feature is available in WhoDB Enterprise Edition'}
                </p>
                <a 
                    href="https://github.com/clidey/whodb/blob/main/ee/README.md" 
                    target="_blank" 
                    rel="noopener noreferrer"
                    className="text-blue-600 dark:text-blue-400 hover:underline"
                >
                    Learn more about Enterprise features →
                </a>
            </div>
        </Card>
    );
};

export const AnalyzeGraphFallback: React.FC = () => {
    return (
        <EEFeatureCard 
            feature="Query Analyzer" 
            description="Visualize query execution plans and optimize database performance with interactive execution graphs."
        />
    );
};

export const LineChartFallback: React.FC = () => {
    return (
        <EEFeatureCard 
            feature="Line Charts" 
            description="Visualize time-series data with interactive line charts."
        />
    );
};

export const PieChartFallback: React.FC = () => {
    return (
        <EEFeatureCard 
            feature="Pie Charts" 
            description="Display categorical data distribution with pie charts."
        />
    );
};

export const ThemeConfigFallback: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    // In CE, just pass through children without custom theming
    return <>{children}</>;
};