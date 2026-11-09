/*
 * Copyright 2026 Clidey, Inc.
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

import { Button, cn, Kbd, ResizableHandle, ResizablePanel, ResizablePanelGroup, SidebarProvider, Tooltip, TooltipContent, TooltipTrigger, useIsMobile, useTheme, IconChevronLeftAlt, IconMenu as Bars3Icon } from '@clidey/ux';
import classNames from "classnames";
import type {FC, ReactNode} from "react";
import {useEffect, useRef, useState} from "react";
import {twMerge} from "tailwind-merge";
import {InternalRoutes, type IInternalRoute} from "../config/routes";
import {useAppDispatch, useAppSelector} from "../store/hooks";
import {SettingsActions} from "../store/settings";
import {Breadcrumb} from "./breadcrumbs";
import {ConnectionContext} from "./connection-context";
import {Loading} from "./loading";
import {Sidebar} from "./sidebar/sidebar";
import {useTranslation} from "@/hooks/use-translation";
import {WhoDBChatIcon} from "./whodb-chat-icon";
import {isMacPlatform} from "@/utils/platform";
import {useSourceSessionMetadata} from "@/hooks/useSourceSessionMetadata";

type IPageProps = {
    wrapperClassName?: string;
    className?: string;
    children: ReactNode;
}

export const Page: FC<IPageProps> = (props) => {
    return <div className={twMerge("flex grow min-w-0 px-8 py-6 flex-col h-full w-full", props.wrapperClassName)}>
        <div className={twMerge("flex flex-row grow flex-wrap gap-sm w-full h-full overflow-y-auto", props.className)}
            data-testid="page-scroll-container">
                {props.children}
        </div>
    </div>
}

type IInternalPageProps = IPageProps & {
    sidebar?: ReactNode | ((onSelect: () => void) => ReactNode);
    children: ReactNode;
    routes?: IInternalRoute[];
    fullHeight?: boolean;
}

const CommandPaletteTrigger: FC = () => {
    const { t } = useTranslation('components/command-palette');

    const handleClick = () => {
        window.dispatchEvent(new CustomEvent('command-palette:open'));
    };

    return (
        <Button
            variant="outline"
            size="sm"
            onClick={handleClick}
            className="gap-2"
            aria-label={t('triggerLabel')}
            data-testid="command-palette-trigger"
        >
            <WhoDBChatIcon name="search" />
            <span className="ce-command-label hidden sm:inline text-xs text-neutral-500 dark:text-neutral-400">{t('triggerLabel')}</span>
            <Kbd className="ce-command-shortcut hidden sm:inline-flex h-5 px-1">
                {isMacPlatform ? '⌘K' : 'Ctrl+K'}
            </Kbd>
        </Button>
    );
};

const KeyboardShortcutsHint: FC = () => {
    const { t } = useTranslation('components/keyboard-shortcuts-help');

    const handleClick = () => {
        // Dispatch a keyboard event to trigger the shortcuts modal
        window.dispatchEvent(new KeyboardEvent('keydown', { key: '?' }));
    };

    return (
        <Tooltip>
            <TooltipTrigger>
                <Button
                    variant="outline"
                    size="sm"
                    onClick={handleClick}
                    className="ce-help-trigger gap-1.5"
                    aria-label={t('showShortcuts')}
                >
                    <WhoDBChatIcon name="help" />
                </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">
                <p>{t('hint')}</p>
            </TooltipContent>
        </Tooltip>
    );
};

const ThemeToggle: FC = () => {
    const {t} = useTranslation('components/page');
    const {theme, setTheme} = useTheme();
    const dark = theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);

    return <Button variant="outline" size="icon" className="ce-theme-trigger" aria-label={t('theme')} onClick={() => { setTheme(dark ? 'light' : 'dark'); }}>
        <WhoDBChatIcon name={dark ? 'sun' : 'moon'} />
    </Button>;
};

/** Renders the app shell with persistent list/detail panes so navigation retains unsaved edits. */
export const InternalPage: FC<IInternalPageProps> = (props) => {
    const { t } = useTranslation('components/page');
    const [detailOpen, setDetailOpen] = useState(true);
    const hasSidebar = Boolean(props.sidebar);
    const isMobile = useIsMobile();
    const detailRef = useRef<HTMLElement>(null);
    const listRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!hasSidebar || !isMobile) return;
        if (detailOpen) detailRef.current?.focus({ preventScroll: true });
        else listRef.current?.querySelector('input')?.focus({ preventScroll: true });
    }, [detailOpen, hasSidebar, isMobile]);
    const isLoggedIn = useAppSelector(state => state.auth.current != null);
    const sidebarOpen = useAppSelector(state => state.settings.sidebarOpen);
    const isTablesPage = props.routes?.at(-1)?.path === InternalRoutes.Dashboard.StorageUnit.path;
    const isExplorePage = props.routes?.at(-1)?.path === InternalRoutes.Dashboard.ExploreStorageUnit.path;
    const isWorkspacePage = [InternalRoutes.RawExecute.path, InternalRoutes.Chat.path, InternalRoutes.Settings?.path, InternalRoutes.ContactUs?.path].includes(props.routes?.at(-1)?.path);
    const dispatch = useAppDispatch();

    // Fetch source session metadata when logged in so Apollo state is ready
    // for source-operator and column-type helpers.
    useSourceSessionMetadata();

    const detail = (
        <main
            ref={detailRef}
            id="main-content"
            className={cn("flex grow flex-wrap gap-sm py-4 content-start relative px-8", props.fullHeight && "min-h-0 !flex-nowrap !px-4 !pt-3 !pb-0", isWorkspacePage && 'ce-workspace-content', hasSidebar && 'min-h-0 flex-1')}
            data-testid="page-content"
            tabIndex={-1}
        >
            {props.children}
        </main>
    );

    return (
        <Container>
            <div className="flex flex-row grow">
                <SidebarProvider open={sidebarOpen} onOpenChange={(open) => dispatch(SettingsActions.setSidebarOpen(open))}>
                    <Sidebar />
                </SidebarProvider>
            </div>
            <Page {...props} wrapperClassName={cn('p-0', isTablesPage && 'ce-tables-shell', isExplorePage && 'ce-explore-shell', isWorkspacePage && 'ce-workspace-shell')}>
                <div className={cn("flex min-h-0 flex-col grow py-6", (isTablesPage || isExplorePage || isWorkspacePage || hasSidebar) && 'ce-app-content', props.fullHeight && "min-h-0 !py-0")}>
                    <div className={cn("flex flex-col gap-1 px-8", (isTablesPage || isExplorePage || isWorkspacePage || hasSidebar) && 'ce-tables-topbar', props.fullHeight && !isWorkspacePage && "px-4 py-5")}>
                        <div className="flex w-full justify-between items-center gap-2">
                            <Button
                                variant="outline"
                                size="icon"
                                className="md:hidden shrink-0"
                                aria-label={t('openNavigation')}
                                data-testid="mobile-nav-toggle"
                                onClick={() => window.dispatchEvent(new CustomEvent('menu:toggle-sidebar'))}
                            >
                                <Bars3Icon className="h-4 w-4" />
                            </Button>
                            <Breadcrumb routes={props.routes ?? []} active={props.routes?.at(-1)} />
                            <div className="flex items-center gap-2 shrink-0">
                                <CommandPaletteTrigger />
                                <KeyboardShortcutsHint />
                                <div data-testid="mode-toggle" role="group" aria-label={t('theme')}>
                                    <ThemeToggle />
                                </div>
                            </div>
                        </div>
                        {!isTablesPage && <ConnectionContext />}
                    </div>
                    {!isLoggedIn ? <div className="flex justify-center items-center h-full">
                        <Loading size="lg" />
                    </div> : props.sidebar ? <div className="ce-master-detail flex min-h-0 flex-1 flex-col" data-detail-open={detailOpen}>
                        {detailOpen && <Button variant="ghost" size="sm" className="m-2 w-fit md:hidden" onClick={() => { setDetailOpen(false); }}>
                            <IconChevronLeftAlt className="size-4" />
                            {t('browseItems')}
                        </Button>}
                        <ResizablePanelGroup direction="horizontal" className="min-h-0 flex-1">
                            <ResizablePanel defaultSize={25} minSize={18} maxSize={40} className="ce-secondary-panel">
                                <div ref={listRef} className="flex h-full min-h-0 flex-col">
                                    <Button variant="ghost" size="sm" className="md:hidden m-2 w-fit" onClick={() => { setDetailOpen(true); }}>
                                        {t('viewContent')}
                                    </Button>
                                    {typeof props.sidebar === 'function' ? props.sidebar(() => { setDetailOpen(true); }) : props.sidebar}
                                </div>
                            </ResizablePanel>
                            <ResizableHandle withHandle className="hidden md:flex" />
                            <ResizablePanel defaultSize={75} className="flex min-w-0 flex-col">
                                {detail}
                            </ResizablePanel>
                        </ResizablePanelGroup>
                    </div> : detail}
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
