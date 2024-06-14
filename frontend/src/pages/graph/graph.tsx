import { FC } from "react";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";

export const GraphPage: FC = () => {
    return <InternalPage routes={[InternalRoutes.Graph]}>
        Graph
    </InternalPage>
}