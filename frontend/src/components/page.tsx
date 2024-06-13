import { FC, ReactNode } from "react";
import { twMerge } from "tailwind-merge";
import { AnimatePresence, motion } from "framer-motion";
import { Sidebar } from "./sidebar";

type IPageProps = {
    className?: string;
    children: ReactNode;
}

export const Page: FC<IPageProps> = (props) => {
    return <div className="flex grow px-8 pt-6 flex-col h-full w-full">
        <AnimatePresence>
            <motion.div className={twMerge("flex flex-row grow flex-wrap gap-2 w-full h-full overflow-y-auto", props.className)}
                initial={{ opacity: 0 }}
                animate={{ opacity: 100, }}
                exit={{ opacity: 0 }}>
                    {props.children}
            </motion.div>
        </AnimatePresence>
    </div>
}

export const InternalPage: FC<IPageProps & { children: ReactNode }> = (props) => {
    return (
        <div className="flex grow h-full w-full">
            <Sidebar />
            <Page {...props}>
                {props.children}
            </Page>
        </div>
    )
}