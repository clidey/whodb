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

import { ModeToggle, SidebarProvider } from "@clidey/ux";
import classNames from "classnames";
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
            <motion.div className={twMerge("flex flex-row grow flex-wrap gap-sm w-full h-full overflow-y-auto", props.className)}
                data-testid="page-scroll-container"
                initial={{ opacity: 0 }}
                animate={{ opacity: 100, transition: { duration: 0.5 } }}
                exit={{ opacity: 0 }}>
                    {props.children}
            </motion.div>
        </AnimatePresence>
    </div>
}

type IInternalPageProps = IPageProps & {
    sidebar?: ReactNode;
    children: ReactNode;
    routes?: IInternalRoute[];
}

export const InternalPage: FC<IInternalPageProps> = (props) => {
    const current = useAppSelector(state => state.auth.current);
    return (
        <Container>
            <div className="flex flex-row grow">
                <SidebarProvider defaultOpen={props.sidebar == null}>
                    <Sidebar />
                </SidebarProvider>
                {props.sidebar && <SidebarProvider>
                    {props.sidebar}
                </SidebarProvider>}
            </div>
            <Page wrapperClassName="p-0" {...props}>
                <div className="flex flex-col grow py-6">
                    <div className="flex w-full justify-between items-center px-8">
                        <Breadcrumb routes={props.routes ?? []} active={props.routes?.at(-1)} />
                        <div data-testid="mode-toggle">
                            <ModeToggle />
                        </div>
                    </div>
                    {
                        current == null
                        ? <Loading />
                            : <div className="flex grow flex-wrap gap-sm py-4 content-start relative px-8"
                                   data-testid="page-content">
                            {props.children}
                        </div>
                    }
                </div>
            </Page>
        </Container>
    )
}

type IContainerProps = {
    children?: ReactNode;
    className?: string;
}

export const Container: FC<IContainerProps> = ({ className, children }) => {
    return  <div className={classNames(className, "flex grow h-full w-full")}>
        {children}
    </div>
}
