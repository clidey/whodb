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
import { Button, Card, CommandItem, EmptyState, Input, Label, ScrollArea, SearchSelect, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Sheet, SheetContent, SheetFooter, toast } from "@clidey/ux";
import { AiChatMessage, GetAiChatQuery, useGetAiChatLazyQuery, useGetAiModelsLazyQuery, useGetAiProvidersLazyQuery } from '@graphql';
import { ArrowTopRightOnSquareIcon, PlusIcon } from "@heroicons/react/24/outline";
import classNames from "classnames";
import { map } from "lodash";
import { cloneElement, FC, KeyboardEventHandler, MouseEvent, ReactElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { v4 } from "uuid";
import { CodeEditor } from "../../components/editor";
import { Icons } from "../../components/icons";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { StorageUnitTable } from "../../components/table";
import { InternalRoutes } from "../../config/routes";
import { reduxStore } from "../../store";
import { AIModelsActions, availableExternalModelTypes } from "../../store/ai-models";
import { HoudiniActions } from "../../store/chat";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ensureModelsArray, ensureModelTypesArray } from "../../utils/ai-models-helper";
import { isEEFeatureEnabled, loadEEComponent } from "../../utils/ee-loader";
import { chooseRandomItems } from "../../utils/functions";
import { chatExamples } from "./examples";
const logoImage = "/images/logo.png";

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
    "Pondering life’s mysteries",
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
    const [showSQL, setShowSQL] = useState(false);

    const handleCodeToggle = useCallback(() => {
        setShowSQL(status => !status);
    }, []);

    return <div className="flex flex-col w-[calc(100%-50px)] group/table-preview gap-2 relative">
        <div className="absolute -top-3 -left-3 opacity-0 group-hover/table-preview:opacity-100 transition-all z-[1]">
            <Button containerClassName="w-8 h-8" className="w-5 h-5" onClick={handleCodeToggle} data-testid="table-preview-code-toggle">
                {cloneElement(showSQL ? Icons.Tables : Icons.Code, {
                    className: "w-6 h-6 stroke-white",
                })}
            </Button>
        </div>
        <div className="flex items-center gap-4 overflow-hidden break-all leading-6 shrink-0 h-full w-full">
            {
                showSQL
                ? <div className="h-[150px] w-full">
                    <CodeEditor value={text} />
                </div>
                :  (data != null && data.Rows.length > 0) || type === "sql:get"
                    ? <div className="h-[250px]">
                        <StorageUnitTable
                            columns={data?.Columns.map(c => c.Name) ?? []}
                            columnTypes={data?.Columns.map(c => c.Type) ?? []}
                            rows={data?.Rows ?? []}
                            disableEdit={true}
                        />
                    </div>
                    : <div className="bg-white/10 text-neutral-800 dark:text-neutral-300 rounded-lg p-2 flex gap-2">
                        Action Executed ({type.toUpperCase().split(":")?.[1]})
                        {Icons.CheckCircle}
                    </div>
            }
        </div>
    </div>
}

type IChatMessage = AiChatMessage & {
    isUserInput?: boolean;
};


export const externalModelTypes = map(availableExternalModelTypes, (model) => ({
    id: model,
    label: model,
    icon: (Icons.Logos as Record<string, ReactElement>)[model],
}));

export const ChatPage: FC = () => {
    const [query, setQuery] = useState("");
    const [addExternalModel, setAddExternalModel] = useState(false);
    const [externalModelType, setExternalModel] = useState<string>(externalModelTypes[0].id);
    const [externalModelToken, setExternalModelToken] = useState<string>();
    const modelType = useAppSelector(state => state.aiModels.current);
    const modelTypesRaw = useAppSelector(state => state.aiModels.modelTypes);
    const modelTypes = ensureModelTypesArray(modelTypesRaw);
    const currentModel = useAppSelector(state => state.aiModels.currentModel);
    const modelsRaw = useAppSelector(state => state.aiModels.models);
    const models = ensureModelsArray(modelsRaw);
    const chats = useAppSelector(state => state.houdini.chats);
    const [modelAvailable, setModelAvailable] = useState(true);
    const [getAiProviders, ] = useGetAiProvidersLazyQuery();
    const dispatch = useAppDispatch();
    const [getAIModels, { loading: getAIModelsLoading }] = useGetAiModelsLazyQuery({
        onError() {
            setModelAvailable(false);
            dispatch(AIModelsActions.setModels([]));
            dispatch(AIModelsActions.setCurrentModel(undefined));
        },
        fetchPolicy: "network-only",
    });
    const [getAIChat, { loading: getAIChatLoading }] = useGetAiChatLazyQuery();
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const schema = useAppSelector(state => state.database.schema);
    const [currentSearchIndex, setCurrentSearchIndex] = useState<number>();

    const loading = useMemo(() => {
        return getAIChatLoading || getAIModelsLoading;
    }, [getAIModelsLoading, getAIChatLoading]);

    const handleAIModelTypeChange = useCallback((item: string) => {
        const modelType = modelTypes.find(model => model.id === item);
        if (modelType == null) {
            return;
        }
        setModelAvailable(true);
        getAIModels({
            variables: {
                providerId: modelType.id,
                modelType: modelType.modelType,
                token: modelType.token,
            },
            onCompleted(data) {
                dispatch(AIModelsActions.setModels(data.AIModel));
                if (data.AIModel.length > 0) {
                    dispatch(AIModelsActions.setCurrentModel(data.AIModel[0]));
                }
            },
        });
    }, [dispatch, getAIModels]);

    const handleAIModelChange = useCallback((item: string) => {
        dispatch(AIModelsActions.setCurrentModel(item));
    }, [dispatch]);

    const handleAIModelRemove = useCallback((_: MouseEvent<HTMLDivElement>, item: string) => {
        if (modelType?.id === item) {
            dispatch(AIModelsActions.setModels([]));
            dispatch(AIModelsActions.setCurrentModel(undefined));
        }
        dispatch(AIModelsActions.removeAIModelType({ id: item }));
    }, [dispatch, modelType?.id]);

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
    }, [chats, currentModel, getAIChat, modelType, query, schema]);

    const handleKeyUp: KeyboardEventHandler<HTMLInputElement> = useCallback((e) => {
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
    }, [chats, currentSearchIndex]);

    const handleSelectExample = useCallback((example: string) => {
        setQuery(example);
    }, []);

    const handleAddExternalModel = useCallback(() => {
        setAddExternalModel(status => !status);
    }, []);

    const handleExternalModelChange = useCallback((item: string) => {
        setExternalModel(item);
    }, []);

    const handleExternalModelSubmit = useCallback(() => {
        dispatch(AIModelsActions.setCurrentModel(undefined));
        dispatch(AIModelsActions.setModels([]));
        getAIModels({
            variables: {
                modelType: externalModelType,
                token: externalModelToken,
            },
            onCompleted(data) {
                dispatch(AIModelsActions.setModels(data.AIModel));
                const id = v4();
                dispatch(AIModelsActions.addAIModelType({
                    id,
                    modelType: externalModelType,
                    token: externalModelToken,
                }));
                dispatch(AIModelsActions.setCurrentModelType({ id }));
                setExternalModel(externalModelTypes[0].id);
                setExternalModelToken("");
                setAddExternalModel(false);
                if (data.AIModel.length > 0) {
                    dispatch(AIModelsActions.setCurrentModel(data.AIModel[0]));
                }
            },
            onError(error) {
                toast.error(`Unable to connect to the model: ${error.message}`);
            },
        });
    }, [getAIModels, externalModelType, externalModelToken, dispatch]);

    const handleOpenDocs = useCallback(() => {
        window.open("https://whodb.com/docs/usage-houdini/what-is-houdini", "_blank");
    }, []);

    useEffect(() => {
        getAiProviders({
            onCompleted(data) {
                const aiProviders = data.AIProviders || [];
                const modelTypesState = ensureModelTypesArray(reduxStore.getState().aiModels.modelTypes);
                const initialModelTypes = modelTypesState.filter(model => {
                const existingModel = aiProviders.find(provider => provider.ProviderId === model.id);
                return existingModel != null || (model.token != null && model.token !== "");
                });

                // Filter out providers that already exist in modelTypes
                const newProviders = aiProviders.filter(provider =>
                !initialModelTypes.some(model => model.id === provider.ProviderId)
                );

                const finalModelTypes = [
                ...newProviders.map(provider => ({
                    id: provider.ProviderId,
                    modelType: provider.Type,
                    isEnvironmentDefined: provider.IsEnvironmentDefined,
                })),
                ...initialModelTypes
                ];

                // Check if current model type exists in final model types
                const currentModelType = reduxStore.getState().aiModels.current;
                if (currentModelType && !finalModelTypes.some(model => model.id === currentModelType.id)) {
                dispatch(AIModelsActions.setCurrentModelType({ id: "" }));
                dispatch(AIModelsActions.setModels([]));
                dispatch(AIModelsActions.setCurrentModel(undefined));
                }

                dispatch(AIModelsActions.setModelTypes(finalModelTypes));
                getAIModels({
                    variables: {
                        providerId: currentModelType?.id,
                        modelType: currentModelType?.modelType ?? "",
                        token: currentModelType?.token ?? "",
                    },
                });
            },
        });

        const modelType = modelTypes[0];
        if (modelType == null || models.length > 0) {
            return;
        }
        handleAIModelTypeChange(modelType.id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const modelTypesDropdownItems = useMemo(() => {
        return modelTypes.filter(modelType => modelType != null && modelType.modelType != null).map(modelType => ({
            id: modelType.id,
            label: modelType.modelType,
            icon: (Icons.Logos as Record<string, ReactElement>)[modelType.modelType.replace("-", "")],
            extra: {
                token: modelType.token,
            }
        }));
    }, [modelTypes]);

    const disableAll = useMemo(() => {
        return models.length === 0 || !modelAvailable;
    }, [modelAvailable, models.length]);

    const disableChat = useMemo(() => {
        return loading || models.length === 0 || !modelAvailable;
    }, [loading, modelAvailable, models.length]);

    const handleClear = useCallback(() => {
        dispatch(HoudiniActions.clear());
    }, []);

    const handleAIProviderChange = useCallback((item: string) => {
        dispatch(AIModelsActions.setCurrentModelType({ id: item }));
        handleAIModelTypeChange(item);
    }, [handleAIModelTypeChange]);

    const modelDropdownItems = useMemo(() => {
        return models.map(model => ({
            id: model,
            label: model,
            icon: (Icons.Logos as Record<string, ReactElement>)[model],
        }));
    }, [models]);

    return (
        <InternalPage routes={[InternalRoutes.Chat]} className="h-full">
            <Sheet open={addExternalModel} onOpenChange={setAddExternalModel}>
                <SheetContent className="max-w-md mx-auto w-full px-8 py-10 flex flex-col gap-4">
                    <div className="flex flex-col gap-4">
                        <div className="text-lg font-semibold mb-2">Add External Model</div>
                        <div className="flex flex-col gap-2">
                            <Label>Model Type</Label>
                            <Select
                                value={externalModelType}
                                onValueChange={handleExternalModelChange}
                            >
                                <SelectTrigger className="w-full" data-testid="external-model-type-select">
                                    <SelectValue placeholder="Select Model Type" />
                                </SelectTrigger>
                                <SelectContent>
                                    {externalModelTypes.map(item => (
                                        <SelectItem key={item.id} value={item.id}>
                                            <span className="flex items-center gap-2">
                                                {item.icon}
                                                {item.label}
                                            </span>
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex flex-col gap-2">
                            <Label>Token</Label>
                            <Input
                                value={externalModelToken ?? ""}
                                onChange={e => setExternalModelToken(e.target.value)}
                                type="password"
                            />
                        </div>
                    </div>
                    <div className="flex items-center gap-2 self-end">
                        <Button
                            onClick={handleAddExternalModel}
                            data-testid="external-model-cancel"
                            variant="secondary"
                        >
                            {Icons.Cancel} Cancel
                        </Button>
                        <Button
                            onClick={handleExternalModelSubmit}
                            disabled={getAIModelsLoading}
                            data-testid="external-model-submit"
                        >
                            {Icons.CheckCircle} Submit
                        </Button>
                    </div>
                    <SheetFooter className="p-0">
                        <div className="text-xs text-neutral-500 mt-4 flex flex-col gap-2">
                            <div className="font-bold">Setup</div>
                            <div>
                                Go to <a href="https://ollama.com/" target="_blank" rel="noopener noreferrer" className="font-semibold underline text-blue-600 hover:text-blue-800">Ollama</a> and follow the installation instructions.
                            </div>
                            <div className="font-semibold">Downloading the Ollama Model</div>
                            <div>
                                Once installed, install the desired model you would like to use. In this guide, we will use <a href="https://ollama.com/library/llama3.1" target="_blank" rel="noopener noreferrer" className="font-semibold underline text-blue-600 hover:text-blue-800">Llama3.1 8b</a>. To install this model, run:
                            </div>
                            <div className="font-mono bg-neutral-100 dark:bg-neutral-900 rounded px-2 py-1 mb-1">
                                ollama run llama3.1
                            </div>
                            <Button icon={Icons.RightArrowUp} variant="secondary" className="w-full mt-2" onClick={handleOpenDocs}>
                                Docs
                                <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                            </Button>
                        </div>
                    </SheetFooter>
                </SheetContent>
            </Sheet>
            <div className="flex flex-col w-full h-full gap-2">
                <div className="flex w-full justify-between">
                    <div className={classNames("flex gap-2", {
                        "opacity-50 pointer-events-none": addExternalModel,
                    })}>
                        <SearchSelect
                            options={modelTypesDropdownItems.map(item => ({
                                value: item.id,
                                label: item.label,
                                icon: item.icon,
                            }))}
                            value={modelType?.id}
                            onChange={id => {
                                const item = modelTypesDropdownItems.find(i => i.id === id);
                                if (item) handleAIProviderChange(item.id);
                            }}
                            placeholder="Select Model Type"
                            side="right"
                            align="start"
                            extraOptions={
                                <CommandItem
                                    key="__add__"
                                    value="__add__"
                                    onSelect={handleAddExternalModel}
                                >
                                    <span className="flex items-center gap-2 text-green-500">
                                        <PlusIcon className="w-4 h-4 stroke-green-500" />
                                        Add another profile
                                    </span>
                                </CommandItem>
                            }
                        />
                        {
                            modelType && <SearchSelect
                                options={modelDropdownItems.map(item => ({
                                    value: item.id,
                                    label: item.label,
                                    icon: item.icon,
                                }))}
                                value={currentModel ? currentModel : undefined}
                                onChange={id => {
                                    const item = modelDropdownItems.find(i => i.id === id);
                                    if (item) handleAIModelChange(item.id);
                                }}
                                placeholder="Select Model"
                                side="right"
                                align="start"
                            />
                        }
                    </div>
                    <div className="flex gap-2">
                        <Button onClick={handleClear} disabled={loading} data-testid="chat-new-chat">
                            {Icons.Refresh} New Chat
                        </Button>
                    </div>
                </div>
                <div className={classNames("flex grow w-full rounded-xl overflow-hidden", {
                    "hidden": disableAll,
                })}>
                    {
                        chats.length === 0
                        ? <div className="flex flex-col justify-center items-center w-full gap-8">
                            <img src={logoImage} alt="clidey logo" className="w-auto h-16" />
                            <div className="flex flex-wrap justify-center items-center gap-4">
                                {
                                    examples.map((example, i) => (
                                        <Card key={`chat-${i}`} className="flex flex-col gap-2 w-[150px] h-[200px] rounded-3xl p-4 text-sm cursor-pointer hover:opacity-80 transition-all"
                                            onClick={() => handleSelectExample(example.description)}>
                                            {example.icon}
                                            {example.description}
                                        </Card>
                                    ))
                                }
                            </div>
                        </div>
                        : <ScrollArea className="h-full w-full py-8 max-h-[calc(80vh-5px)]" ref={scrollContainerRef}>
                            <div className="flex justify-center w-full">
                                <div className="flex w-[max(65%,450px)] flex-col gap-2">
                                    {
                                        chats.map((chat, i) => {
                                            if (chat.Type === "message" || chat.Type === "text") {
                                                return <div key={`chat-${i}`} className={classNames("flex gap-4 overflow-hidden break-words leading-6 shrink-0 relative", {
                                                    "self-end": chat.isUserInput,
                                                    "self-start": !chat.isUserInput,
                                                })}>
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput
                                                        ? <img src={logoImage} alt="clidey logo" className="w-auto h-6 mt-2" />
                                                        : <div className="pl-4" />}
                                                    <div className={classNames("text-neutral-800 dark:text-neutral-300 px-4 py-2 rounded-xl whitespace-pre-wrap", {
                                                        "bg-neutral-600/5 dark:bg-[#2C2F33]": chat.isUserInput,
                                                        "-ml-2": !chat.isUserInput && chats[i-1]?.isUserInput,
                                                    })}>
                                                        {chat.Text}
                                                    </div>
                                                </div>
                                            } else if (chat.Type === "error") {
                                                return <div key={`chat-${i}`} className="flex items-center gap-4 overflow-hidden break-words leading-6 shrink-0 self-start">
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput
                                                        ? <img src={logoImage} alt="clidey logo" className="w-auto h-6" />
                                                        : <div className="pl-4" />}
                                                    <div className="text-red-800 dark:text-red-300 px-4 py-2 rounded-lg">
                                                        {chat.Text}
                                                    </div>
                                                </div>
                                            } else if (isEEFeatureEnabled('dataVisualization') && (chat.Type === "sql:pie-chart" || chat.Type === "sql:line-chart")) {
                                                return <div key={`chat-${i}`} className="flex items-center self-start">
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput && <img src={logoImage} alt="clidey logo" className="w-auto h-6" />}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:pie-chart" && PieChart && <PieChart columns={chat.Result?.Columns.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} />}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:line-chart" && LineChart && <LineChart columns={chat.Result?.Columns.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} />}
                                                </div>
                                            }
                                            return <div key={`chat-${i}`} className="flex gap-4 w-full overflow-hidden pt-4 pr-9">
                                                {!chat.isUserInput && chats[i-1]?.isUserInput
                                                    ? <img src={logoImage} alt="clidey logo" className="w-auto h-6" />
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
                        </ScrollArea>
                    }
                </div>
                {
                    (!modelAvailable || models.length === 0) &&
                    <EmptyState title="No Model Available" description="Please choose an available model to start chatting with your data." icon={Icons.Sparkles} />
                }
                <div className={classNames("relative w-full", {
                    "opacity-80": disableChat,
                    "opacity-10": disableAll,
                })}>
                    <div className={classNames("absolute right-2 top-1/2 -translate-y-1/2 z-10 backdrop-blur-lg rounded-full cursor-pointer hover:scale-110 transition-all", {
                        "opacity-50": loading,
                    })} onClick={loading ? undefined : handleSubmitQuery}>
                        {cloneElement(Icons.ArrowUp, {
                            className: "w-5 h-5 stroke-neutral-800 dark:stroke-neutral-300",
                        })}
                    </div>
                    <Input value={query} onChange={e => setQuery(e.target.value)} placeholder="Talk to me..." onSubmit={handleSubmitQuery}
                        disabled={disableChat} onKeyUp={handleKeyUp} />
                </div>
            </div>
        </InternalPage>
    )
}
