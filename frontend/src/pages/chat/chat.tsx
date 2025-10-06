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

import { isEEMode } from "@/config/ee-imports";
import { Alert, AlertDescription, AlertTitle, Button, Card, cn, EmptyState, Input, toast, toTitleCase, Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@clidey/ux";
import { AiChatMessage, GetAiChatQuery, useGetAiChatLazyQuery } from '@graphql';
import {
    ArrowUpCircleIcon,
    CheckCircleIcon,
    CodeBracketIcon,
    SparklesIcon,
    TableCellsIcon,
    CommandLineIcon
} from "../../components/heroicons";
import classNames from "classnames";
import { cloneElement, FC, KeyboardEventHandler, useCallback, useMemo, useRef, useState } from "react";
import logoImage from "../../../public/images/logo.png";
import { AIProvider, useAI } from "../../components/ai";
import { CodeEditor } from "../../components/editor";
import { ErrorState } from "../../components/error-state";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { StorageUnitTable } from "../../components/table";
import { extensions } from "../../config/features";
import { InternalRoutes } from "../../config/routes";
import { HoudiniActions } from "../../store/chat";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ScratchpadActions } from "../../store/scratchpad";
import { isEEFeatureEnabled, loadEEComponent } from "../../utils/ee-loader";
import { chooseRandomItems } from "../../utils/functions";
import { databaseSupportsScratchpad } from "../../utils/database-features";
import { useNavigate } from "react-router-dom";
import { chatExamples } from "./examples";

// Lazy load chart components if EE is enabled
const LineChart = isEEFeatureEnabled('dataVisualization') ? loadEEComponent(
    () => import('@ee/components/charts/line-chart').then(m => ({ default: m.LineChart })),
    () => null
) : () => null;

const PieChart = isEEFeatureEnabled('dataVisualization') ? loadEEComponent(
    () => import('@ee/components/charts/pie-chart').then(m => ({ default: m.PieChart })),
    () => null
) : () => null;

const thinkingPhrases = [
    "Thinking",
    "Pondering life's mysteries",
    "Consulting the cloud oracles",
    "Googling furiously (just kidding)",
    "Aligning the neural networks",
    "Making it up as I go (shh)",
    "Counting virtual sheep",
    "Channeling Einstein",
    "Tuning my algorithms",
    "Drinking a byte of coffee",
    "Running in circles virtually.",
    "Pretending to be busy",
    "Loading witty comeback",
    "Downloading some wisdom",
    "Cooking up some data stew",
    "Doing AI things™",
    "Hacking the mainframe (for fun)",
    "Staring into the digital abyss",
    "Flipping a quantum coin",
    "Reading your mind (ethically)",
    "Sharpening my sarcasm",
    "Checking my vibes",
    "Simulating deep thought",
    "Rewiring my circuits",
    "Polishing my crystal processor"
  ];
  

type TableData = GetAiChatQuery["AIChat"][0]["Result"];

const TablePreview: FC<{ type: string, data: TableData, text: string }> = ({ type, data, text }) => {
    const dispatch = useAppDispatch();
    const [showSQL, setShowSQL] = useState(false);
    const [showScratchpadDialog, setShowScratchpadDialog] = useState(false);
    const [selectedPage, setSelectedPage] = useState<string>("new");
    const [newPageName, setNewPageName] = useState<string>("");
    const navigate = useNavigate();
    const current = useAppSelector(state => state.auth.current);
    const { pages, activePageId } = useAppSelector(state => state.scratchpad);

    const handleCodeToggle = useCallback(() => {
        setShowSQL(status => !status);
    }, []);

    // Create page options excluding current page
    const pageOptions = useMemo(() => {
        return [
            ...pages.map(page => ({ value: page.id, label: page.name })),
            { value: "new", label: "Create new page" }
        ];
    }, [pages, activePageId]);

    const handleMoveToScratchpad = useCallback(() => {
        if (!databaseSupportsScratchpad(current?.Type)) {
            toast.error("Scratchpad is not supported for this database type");
            return;
        }
        // Initialize scratchpad if needed
        if (pages.length === 0) {
            dispatch(ScratchpadActions.ensurePagesHaveCells());
        }
        setShowScratchpadDialog(true);
    }, [current?.Type, pages.length, dispatch]);

    const handleScratchpadConfirm = useCallback(() => {
        if (selectedPage === "new") {
            // Create new page with the query
            const pageName = newPageName.trim() || `Page ${pages.length + 1}`;
            dispatch(ScratchpadActions.addPage({ name: pageName, initialQuery: text }));
            // Navigate to scratchpad - the new page will be created with the query
            navigate(InternalRoutes.RawExecute.path, {
                state: {
                    targetPage: "new"
                }
            });
        } else {
            // Add to existing page and set it as active
            dispatch(ScratchpadActions.addCellToPageAndActivate({ 
                pageId: selectedPage, 
                initialQuery: text 
            }));
            // Navigate to scratchpad and highlight the target page
            navigate(InternalRoutes.RawExecute.path, {
                state: {
                    targetPage: selectedPage
                }
            });
        }
        setShowScratchpadDialog(false);
        setSelectedPage("new");
        setNewPageName("");
        toast.success("Query moved to scratchpad");
    }, [navigate, text, selectedPage, newPageName, pages.length, dispatch]);

    const previewResult = useMemo(() => {
        if (data == null || data.Rows.length === 0) {
            return "No data was returned.";
        }
        return type.toUpperCase().split(":")?.[1];
    }, [data, type]);

    const canMoveToScratchpad = useMemo(() => {
        return databaseSupportsScratchpad(current?.Type) && type.startsWith("sql:");
    }, [current?.Type, type]);

    return <div className="flex flex-col w-[calc(100%-50px)] group/table-preview">
        <div className="opacity-0 group-hover/table-preview:opacity-100 transition-all z-[1] flex gap-1 -transalte-y-full h-0">
            <Button onClick={handleCodeToggle} data-testid="icon-button" variant="outline">
                {cloneElement(showSQL ? <TableCellsIcon className="w-6 h-6" /> : <CodeBracketIcon className="w-6 h-6" />, {
                    className: "w-6 h-6",
                })}
            </Button>
            {canMoveToScratchpad && (
                <Button 
                    variant="outline"
                    onClick={handleMoveToScratchpad} 
                    data-testid="icon-button"
                    title="Move to Scratchpad"
                >
                    <CommandLineIcon className="w-6 h-6" />
                </Button>
            )}
        </div>
        <div className="flex items-center gap-lg overflow-hidden break-all leading-6 shrink-0 h-full w-full">
            {
                showSQL
                ? <div className="h-[150px] w-full">
                    <CodeEditor value={text} />
                </div>
                :  (data != null && data.Rows.length > 0) || type === "sql:get"
                    ? <div className="h-[250px] w-full">
                        <StorageUnitTable
                            columns={data?.Columns.map(c => c.Name) ?? []}
                            columnTypes={data?.Columns.map(c => c.Type) ?? []}
                            rows={data?.Rows ?? []}
                            disableEdit={true}
                        />
                    </div>
                    : <Alert title="Action Executed" className="w-fit">
                        <CheckCircleIcon className="w-4 h-4" />
                        <AlertTitle>Action Executed</AlertTitle>
                        <AlertDescription>
                            {previewResult}
                        </AlertDescription>
                    </Alert>
            }
        </div>
        
        <Dialog open={showScratchpadDialog} onOpenChange={setShowScratchpadDialog}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Move to Scratchpad</DialogTitle>
                    <DialogDescription>
                        Choose which scratchpad page to add this query to.
                    </DialogDescription>
                </DialogHeader>
                <div className="py-4 space-y-4">
                    <div>
                        <label className="text-sm font-medium mb-2 block">Select a page</label>
                        <Select value={selectedPage} onValueChange={setSelectedPage}>
                            <SelectTrigger className="w-full">
                                <SelectValue placeholder="Choose a page..." />
                            </SelectTrigger>
                            <SelectContent>
                                {pageOptions.map((option) => (
                                    <SelectItem key={option.value} value={option.value}>
                                        {option.label}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                    {selectedPage === "new" && (
                        <div>
                            <label className="text-sm font-medium mb-2 block">New page name</label>
                            <Input
                                value={newPageName}
                                onChange={(e) => setNewPageName(e.target.value)}
                                placeholder="Enter page name"
                            />
                        </div>
                    )}
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => {
                        setShowScratchpadDialog(false);
                        setSelectedPage("new");
                        setNewPageName("");
                    }}>
                        Cancel
                    </Button>
                    <Button onClick={handleScratchpadConfirm}>
                        Move to Scratchpad
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    </div>
}

type IChatMessage = AiChatMessage & {
    isUserInput?: boolean;
};

export const ChatPage: FC = () => {
    const [query, setQuery] = useState("");
    const chats = useAppSelector(state => state.houdini.chats);
    const [getAIChat, { loading: getAIChatLoading }] = useGetAiChatLazyQuery();
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const schema = useAppSelector(state => state.database.schema);
    const [currentSearchIndex, setCurrentSearchIndex] = useState<number>();

    const dispatch = useAppDispatch();

    const aiState = useAI();
    const { modelType, currentModel, modelAvailable, models } = aiState;

    const loading = useMemo(() => {
        return getAIChatLoading;
    }, [getAIChatLoading]);

    const examples = useMemo(() => {
        return chooseRandomItems(chatExamples);
    }, []);

    const handleSubmitQuery = useCallback(() => {
        const sanitizedQuery = query.trim();
        if (modelType == null || sanitizedQuery.length === 0) {
            return;
        }

        dispatch(HoudiniActions.addChatMessage({ Type: "message", Text: sanitizedQuery, isUserInput: true, }));
        setTimeout(() => {
            if (scrollContainerRef.current != null) {
                scrollContainerRef.current.scroll({
                    top: scrollContainerRef.current.scrollHeight,
                    behavior: "smooth",
                });
            }
        }, 250);
        getAIChat({
            variables: {
                modelType: modelType.modelType,
                token: modelType.token,
                query: sanitizedQuery,
                model: currentModel ?? "",
                previousConversation: chats.map(chat => `${chat.isUserInput ? "<User>" : "<System>"}${chat.Text}${chat.isUserInput ? "</User>" : "</System>"}`).join("\n"),
                schema,
            },
            onCompleted(data) {
                const systemChats: IChatMessage[] = data.AIChat.map(chat => {
                    if (chat.Type.startsWith("sql")) {
                        return {
                            Type: chat.Type,
                            Text: chat.Text,
                            Result: chat.Result as AiChatMessage["Result"],
                        }
                    }
                    return {
                        Type: chat.Type,
                        Text: chat.Text,
                    }
                });
                for (const systemChat of systemChats) {
                    dispatch(HoudiniActions.addChatMessage(systemChat));
                }
                setTimeout(() => {
                    if (scrollContainerRef.current != null) {
                        scrollContainerRef.current.scroll({
                            top: scrollContainerRef.current.scrollHeight,
                            behavior: "smooth",
                        });
                    }
                }, 250);
            },
            onError(error) {
                toast.error("Unable to query. Try again. "+error.message);
            },
        });
        setQuery("");
    }, [chats, currentModel, getAIChat, modelType, query, schema, dispatch]);

    const disableChat = useMemo(() => {
        return loading || models.length === 0 || !modelAvailable || query.trim().length === 0;
    }, [loading, modelAvailable, models.length, query]);

    const handleKeyUp: KeyboardEventHandler<HTMLInputElement> = useCallback((e) => {
        if (e.key === "Enter") {
            if (query.trim().length > 0 && !disableChat) {
                handleSubmitQuery();
            }
            return;
        }
        if (e.key === "ArrowUp") {
          const foundSearchIndex = currentSearchIndex != null ? currentSearchIndex - 1 : chats.length - 1;
          let searchIndex = foundSearchIndex;

          while (searchIndex >= 0) {
            if (chats[searchIndex].isUserInput) {
              setCurrentSearchIndex(searchIndex);
              setQuery(chats[searchIndex].Text);
              return;
            }
            searchIndex--;
          }

          if (currentSearchIndex !== chats.length - 1) {
            searchIndex = chats.length - 1;
            while (searchIndex > foundSearchIndex) {
              if (chats[searchIndex].isUserInput) {
                setCurrentSearchIndex(searchIndex);
                setQuery(chats[searchIndex].Text);
                return;
              }
              searchIndex--;
            }
          }
        }
    }, [chats, currentSearchIndex, query, handleSubmitQuery, disableChat]);

    const handleSelectExample = useCallback((example: string) => {
        setQuery(example);
    }, []);

    const handleClear = useCallback(() => {
        dispatch(HoudiniActions.clear());
    }, [dispatch]);

    const disableAll = useMemo(() => {
        return models.length === 0 || !modelAvailable;
    }, [modelAvailable, models.length]);

    return (
        <InternalPage routes={[InternalRoutes.Chat]} className="h-full">
            <div className="flex flex-col w-full h-full gap-2">
                <AIProvider 
                    {...aiState}
                    onClear={handleClear}
                />
                <div className={classNames("flex grow w-full rounded-xl overflow-hidden", {
                    "hidden": disableAll,
                })}>
                    {
                        chats.length === 0
                        ? <div className="flex flex-col justify-center items-center w-full gap-8">
                            {extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-16" />}
                            <div className="flex flex-wrap justify-center items-center gap-4">
                                {
                                    examples.map((example, i) => (
                                        <Card key={`chat-${i}`} className="flex flex-col gap-sm w-[250px] h-[120px] p-4 text-sm cursor-pointer hover:opacity-80 transition-all"
                                            onClick={() => handleSelectExample(example.description)}>
                                            {example.icon}
                                            {example.description}
                                        </Card>
                                    ))
                                }
                            </div>
                        </div>
                        : <div className="h-full w-full py-8 max-h-[calc(75vh-25px)] overflow-y-auto" ref={scrollContainerRef}>
                            <div className="flex justify-center w-full h-full">
                                <div className="flex w-full flex-col gap-2">
                                    {
                                        chats.map((chat, i) => {
                                            if (chat.Type === "message" || chat.Type === "text") {
                                                return <div key={`chat-${i}`} className={classNames("flex gap-lg overflow-hidden break-words leading-6 shrink-0 relative", {
                                                    "self-end ml-3": chat.isUserInput,
                                                    "self-start": !chat.isUserInput,
                                                })}>
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput
                                                        ? extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8 mt-2" />
                                                        : <div className="pl-4" />}
                                                    <p className={classNames("px-4 py-2 rounded-xl whitespace-pre-wrap", {
                                                        "bg-neutral-600/5 dark:bg-[#2C2F33]": chat.isUserInput,
                                                        "-ml-2": !chat.isUserInput && chats[i-1]?.isUserInput,
                                                    })} data-input-message={chat.isUserInput ? "user" : "system"}>
                                                        {chat.Text}
                                                    </p>
                                                </div>
                                            } else if (chat.Type === "error") {
                                                return (
                                                    <div key={`chat-${i}`} className="flex gap-lg overflow-hidden break-words leading-6 shrink-0 self-start pt-6">
                                                        {!chat.isUserInput && chats[i-1]?.isUserInput
                                                            ? extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />
                                                            : <div className="pl-4" />}
                                                        <ErrorState error={toTitleCase(chat.Text.replaceAll("ERROR: ", ""))} />
                                                    </div>
                                                );
                                            } else if (isEEFeatureEnabled('dataVisualization') && (chat.Type === "sql:pie-chart" || chat.Type === "sql:line-chart")) {
                                                return <div key={`chat-${i}`} className="flex items-center self-start">
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput && (extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />)}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:pie-chart" && PieChart && <PieChart columns={chat.Result?.Columns.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} />}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:line-chart" && LineChart && <LineChart columns={chat.Result?.Columns.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} />}
                                                </div>
                                            }
                                            return <div key={`chat-${i}`} className="flex gap-lg w-full pt-4">
                                                {!chat.isUserInput && chats[i-1]?.isUserInput
                                                    ? (extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />)
                                                    : <div className="pl-4" />}
                                                <TablePreview type={chat.Type} text={chat.Text} data={chat.Result} />
                                            </div>
                                        })
                                    }
                                    { loading &&  <div className="flex w-full mt-4">
                                        <Loading loadingText={isEEMode ? thinkingPhrases[0] : chooseRandomItems(thinkingPhrases)[0]} size="sm" />
                                    </div> }
                                </div>
                            </div>
                        </div>
                    }
                </div>
                {
                    (!modelAvailable || models.length === 0) &&
                    <EmptyState title="No Model Available" description="Please choose an available model to start chatting with your data." icon={<SparklesIcon className="w-16 h-16" data-testid="empty-state-sparkles-icon" />} />
                }
                <div className={classNames("flex justify-between items-center gap-2", {
                    "opacity-80": disableChat,
                    "opacity-10": disableAll,
                })}>
                    <Input
                        value={query}
                        onChange={e => setQuery(e.target.value)}
                        placeholder="Type your message here..."
                        onSubmit={handleSubmitQuery}
                        disabled={disableAll}
                        onKeyUp={handleKeyUp}
                    />
                    <Button tabIndex={0} onClick={loading ? undefined : handleSubmitQuery} className={cn("rounded-full", {
                        "opacity-50": loading,
                    })} disabled={disableChat} variant={disableChat ? "secondary" : undefined}>
                        <ArrowUpCircleIcon className="w-8 h-8" />
                    </Button>
                </div>
            </div>
        </InternalPage>
    )
}
