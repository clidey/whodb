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

import {skipToken, useQuery} from "@apollo/client/react";
import type { FC } from "react";
import { GetAiModelsDocument } from '@graphql';
import { Loading } from "../../components/loading";
import { Navigate } from "react-router-dom";
import { InternalRoutes } from "../../config/routes";
import { getSurfaceFallbackPath } from "../../config/route-registry";
import { useSourceContract } from "../../hooks/useSourceContract";
import { InternalPage } from "../../components/page";
import { useAppSelector } from "../../store/hooks";
import { availableInternalModelTypes } from "../../store/ai-models";
import { hasComponent } from "../../config/component-registry";

export const NavigateToDefault: FC = () => {
    const currentType = useAppSelector(state => state.auth.current?.Type);
    const { supportsChat } = useSourceContract(currentType);

    if (hasComponent('sql-agent')) {
        return <Navigate to={InternalRoutes.Chat.path} />
    }

    const defaultModelType = availableInternalModelTypes[0];
    const aiModelsQueryOptions = currentType && supportsChat && defaultModelType
        ? {
            variables: {
                modelType: defaultModelType,
            }
        }
        : skipToken;
    const { data, error } = useQuery(GetAiModelsDocument, aiModelsQueryOptions);

    if (!supportsChat ||  error != null) {
        return <Navigate to={getSurfaceFallbackPath()} />
    }

    if (data?.AIModel != null) {
        if (data.AIModel.length > 0) {
            return <Navigate to={InternalRoutes.Chat.path} />
        }
        return <Navigate to={getSurfaceFallbackPath()} />
    }

    return <InternalPage>
        <Loading />
    </InternalPage>
  }
