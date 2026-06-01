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

import type { ReactNode} from "react";
import { useMemo } from "react";
import { CircleStackIcon, UsersIcon, UserGroupIcon, InformationCircleIcon, PresentationChartLineIcon, ShoppingBagIcon, BuildingStorefrontIcon, BanknotesIcon, PresentationChartBarIcon } from "../../components/heroicons";
import { useTranslation } from "@/hooks/use-translation";

export interface ChatExample {
    icon: ReactNode;
    description: string;
}

const exampleIcons: ReactNode[] = [
    <CircleStackIcon key="circle-stack" className="w-4 h-4" />,
    <UsersIcon key="users" className="w-4 h-4" />,
    <UserGroupIcon key="user-group-0" className="w-4 h-4" />,
    <InformationCircleIcon key="info-circle" className="w-4 h-4" />,
    <PresentationChartLineIcon key="chart-line-0" className="w-4 h-4" />,
    <PresentationChartLineIcon key="chart-line-1" className="w-4 h-4" />,
    <UserGroupIcon key="user-group-1" className="w-4 h-4" />,
    <ShoppingBagIcon key="shopping-bag-0" className="w-4 h-4" />,
    <ShoppingBagIcon key="shopping-bag-1" className="w-4 h-4" />,
    <BuildingStorefrontIcon key="building-storefront" className="w-4 h-4" />,
    <UserGroupIcon key="user-group-2" className="w-4 h-4" />,
    <UserGroupIcon key="user-group-3" className="w-4 h-4" />,
    <UserGroupIcon key="user-group-4" className="w-4 h-4" />,
    <BanknotesIcon key="banknotes-0" className="w-4 h-4" />,
    <BanknotesIcon key="banknotes-1" className="w-4 h-4" />,
    <BanknotesIcon key="banknotes-2" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-0" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-1" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-2" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-3" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-4" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-5" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-6" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-7" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-8" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-9" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-10" className="w-4 h-4" />,
    <PresentationChartBarIcon key="chart-bar-11" className="w-4 h-4" />,
];

export const useChatExamples = (): ChatExample[] => {
    const { t } = useTranslation("pages/chat");

    return useMemo(() => {
        return exampleIcons.map((icon, index) => ({
            icon,
            description: t(`example${index}`),
        }));
    }, [t]);
};

export const chatExamples: ChatExample[] = exampleIcons.map((icon, index) => ({
    icon,
    description: `example${index}`,
}));