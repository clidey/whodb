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

import { FC } from "react";
import { DatabaseType, useGetAiModelsQuery } from "../../generated/graphql";
import { Loading } from "../../components/loading";
import { Navigate } from "react-router-dom";
import { InternalRoutes } from "../../config/routes";
import { InternalPage } from "../../components/page";
import { useAppSelector } from "../../store/hooks";
import { isNoSQL } from "../../utils/functions";
import { availableInternalModelTypes } from "../../store/database";

export const NavigateToDefault: FC = () => {
    const current = useAppSelector(state => state.auth.current);
    const { data, error } = useGetAiModelsQuery({
        variables: {
            modelType: availableInternalModelTypes[0],
        }
    });

    if (isNoSQL(current?.Type as DatabaseType) ||  error != null) {
        return <Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />
    }

    if (data?.AIModel != null) {
        if (data.AIModel.length > 0) {
            return <Navigate to={InternalRoutes.Chat.path} />
        }
        return <Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />
    }

    return <InternalPage>
        <Loading />
    </InternalPage>
  }