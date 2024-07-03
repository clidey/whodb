import { AnimatePresence, motion } from "framer-motion";
import { FC, ReactNode } from "react";
import { twMerge } from "tailwind-merge";
import { IInternalRoute } from "../config/routes";
import { useAppSelector } from "../store/hooks";
import { Breadcrumb } from "./breadcrumbs";
import { Loading } from "./loading";
import { Sidebar } from "./sidebar/sidebar";

type IPageProps = {
    wrapperClassName?: string;
    className?: string;
    children: ReactNode;
}

export const Page: FC<IPageProps> = (props) => {
    return <div className={twMerge("flex grow px-8 py-6 flex-col h-full w-full", props.wrapperClassName)}>
        <AnimatePresence>
            <motion.div className={twMerge("flex flex-row grow flex-wrap gap-2 w-full h-full overflow-y-auto", props.className)}
                initial={{ opacity: 0 }}
                animate={{ opacity: 100, transition: { duration: 0.5 } }}
                exit={{ opacity: 0 }}>
                    {props.children}
            </motion.div>
        </AnimatePresence>
    </div>
}

type IInternalPageProps = IPageProps & {
    children: ReactNode;
    routes?: IInternalRoute[];
}

export const InternalPage: FC<IInternalPageProps> = (props) => {
    const current = useAppSelector(state => state.auth.current);

    return (
        <div className="flex grow h-full w-full">
            <Sidebar />
            <Page wrapperClassName="p-0" {...props}>
                <div className="flex flex-col grow py-6">
                    <div className="px-4 sticky z-10 top-2 left-4 bg-white w-fit rounded-xl py-2">
                        <Breadcrumb routes={props.routes ?? []} active={props.routes?.at(-1)} />
                    </div>
                    {
                        current == null
                        ? <Loading />
                        : <div className="flex grow flex-wrap gap-2 py-4 content-start relative px-8">
                            {props.children}
                        </div>
                    }
                </div>
            </Page>
        </div>
    )
}