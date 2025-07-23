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

import { AnimatePresence, motion } from "framer-motion";
import { FC, ReactNode, useCallback } from "react";
import { twMerge } from "tailwind-merge";
import { IInternalRoute } from "../config/routes";
import { GlobalActions } from "../store/global";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import { Breadcrumb } from "./breadcrumbs";
import { ClassNames } from "./classes";
import { Icons } from "./icons";
import { Loading } from "./loading";
import { Sidebar } from "./sidebar/sidebar";
import classNames from "classnames";

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
    const darkModeEnabled = useAppSelector(state => state.global.theme === "dark");
    const dispatch = useAppDispatch();

    const handleDarkModeToggle = useCallback(() => {
        dispatch(GlobalActions.setTheme(darkModeEnabled ? "light" : "dark"));
    }, [dispatch, darkModeEnabled]);

    return (
        <Container>
            <Sidebar />
            <Page wrapperClassName="p-0" {...props}>
                <div className="flex flex-col grow py-6">
                    <div className="flex justify-between items-center">
                        <div className="sticky z-10 top-2 left-4 w-fit rounded-xl transition-all">
                            <Breadcrumb routes={props.routes ?? []} active={props.routes?.at(-1)} />
                        </div>
                        <div className={classNames("flex gap-2 items-center mr-8 cursor-pointer rounded-full", ClassNames.Text, ClassNames.Hover)} onClick={handleDarkModeToggle}>
                            {darkModeEnabled ? Icons.Sun : Icons.Moon }
                        </div>
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
        </Container>
    )
}

type IContainerProps = {
    children?: ReactNode;
    className?: string;
}

export const Container: FC<IContainerProps> = ({ className, children }) => {
    return  <div className={classNames(className, "flex grow h-full w-full", ClassNames.Background)}>
        {children}
    </div>
}
