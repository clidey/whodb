import { FC } from "react";
import { DatabaseType, useGetAiModelsQuery } from "../../generated/graphql";
import { Loading } from "../../components/loading";
import { Navigate } from "react-router-dom";
import { InternalRoutes } from "../../config/routes";
import { InternalPage } from "../../components/page";
import { useAppSelector } from "../../store/hooks";
import { isNoSQL } from "../../utils/functions";

export const NavigateToDefault: FC = () => {
    const current = useAppSelector(state => state.auth.current);
    const { data, error } = useGetAiModelsQuery();

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