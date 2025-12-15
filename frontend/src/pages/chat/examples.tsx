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

import { ReactNode, useMemo } from "react";
import { CircleStackIcon, UsersIcon, UserGroupIcon, InformationCircleIcon, PresentationChartLineIcon, ShoppingBagIcon, BuildingStorefrontIcon, BanknotesIcon, PresentationChartBarIcon } from "../../components/heroicons";
import { useTranslation } from "@/hooks/use-translation";

export interface ChatExample {
    icon: ReactNode;
    description: string;
}

const exampleIcons: ReactNode[] = [
    <CircleStackIcon className="w-4 h-4" />,
    <UsersIcon className="w-4 h-4" />,
    <UserGroupIcon className="w-4 h-4" />,
    <InformationCircleIcon className="w-4 h-4" />,
    <PresentationChartLineIcon className="w-4 h-4" />,
    <PresentationChartLineIcon className="w-4 h-4" />,
    <UserGroupIcon className="w-4 h-4" />,
    <ShoppingBagIcon className="w-4 h-4" />,
    <ShoppingBagIcon className="w-4 h-4" />,
    <BuildingStorefrontIcon className="w-4 h-4" />,
    <UserGroupIcon className="w-4 h-4" />,
    <UserGroupIcon className="w-4 h-4" />,
    <UserGroupIcon className="w-4 h-4" />,
    <BanknotesIcon className="w-4 h-4" />,
    <BanknotesIcon className="w-4 h-4" />,
    <BanknotesIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
    <PresentationChartBarIcon className="w-4 h-4" />,
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