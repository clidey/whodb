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

import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
    Button,
    cn,
    CommandItem,
    Input,
    Label,
    SearchSelect,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Sheet,
    SheetContent,
    SheetFooter,
    toast
} from "@clidey/ux";
import map from "lodash/map";
import { FC, ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { v4 } from "uuid";
import { useGetAiModelsLazyQuery, useGetAiProvidersLazyQuery } from "../generated/graphql";
import { reduxStore } from "../store";
import { AIModelsActions, availableExternalModelTypes } from "../store/ai-models";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import { ensureModelsArray, ensureModelTypesArray } from "../utils/ai-models-helper";
import { ExternalLink } from "../utils/external-links";
import {
    ArrowPathIcon,
    ArrowTopRightOnSquareIcon,
    CheckCircleIcon,
    LockClosedIcon,
    PlusCircleIcon,
    TrashIcon,
    XMarkIcon
} from "./heroicons";
import { Icons } from "./icons";

export const externalModelTypes = map(availableExternalModelTypes, (model) => ({
    id: model,
    label: model,
    icon: (Icons.Logos as Record<string, ReactElement>)[model],
}));

export const useAI = () => {
    const modelType = useAppSelector(state => state.aiModels.current);
    const currentModel = useAppSelector(state => state.aiModels.currentModel);
    const modelTypesRaw = useAppSelector(state => state.aiModels.modelTypes);
    const modelTypes = ensureModelTypesArray(modelTypesRaw);
    const modelsRaw = useAppSelector(state => state.aiModels.models);
    const models = ensureModelsArray(modelsRaw);
    const [modelAvailable, setModelAvailable] = useState(true);

    const dispatch = useAppDispatch();

    const [getAiProviders, { loading }] = useGetAiProvidersLazyQuery();
    const [getAIModels, { loading: getAIModelsLoading }] = useGetAiModelsLazyQuery({
        onError() {
            setModelAvailable(false);
            dispatch(AIModelsActions.setModels([]));
            dispatch(AIModelsActions.setCurrentModel(undefined));
        },
        fetchPolicy: "network-only",
    });

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
    }, [dispatch, getAIModels, modelTypes]);

    const handleAIModelChange = useCallback((item: string) => {
        dispatch(AIModelsActions.setCurrentModel(item));
    }, [dispatch]);

    const handleAIModelRemove = useCallback((_: any, item: string) => {
        if (modelType?.id === item) {
            dispatch(AIModelsActions.setModels([]));
            dispatch(AIModelsActions.setCurrentModel(undefined));
        }
        dispatch(AIModelsActions.removeAIModelType({ id: item }));
    }, [dispatch, modelType?.id]);

    const handleAIProviderChange = useCallback((item: string) => {
        dispatch(AIModelsActions.setCurrentModelType({ id: item }));
        handleAIModelTypeChange(item);
    }, [handleAIModelTypeChange]);

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
                isEnvironmentDefined: modelType.isEnvironmentDefined,
            }
        }));
    }, [modelTypes]);

    const modelDropdownItems = useMemo(() => {
        return models.map(model => ({
            id: model,
            label: model,
            icon: (Icons.Logos as Record<string, ReactElement>)[model],
        }));
    }, [models]);

    return {
        modelType,
        modelTypes,
        currentModel,
        models,
        loading,
        getAiProviders,
        getAIModels,
        getAIModelsLoading,
        modelAvailable,
        handleAIModelTypeChange,
        handleAIModelChange,
        handleAIModelRemove,
        handleAIProviderChange,
        modelTypesDropdownItems,
        modelDropdownItems,
    }
}

export const AIProvider: FC<ReturnType<typeof useAI> & { 
    disableNewChat?: boolean;
    onClear?: () => void;
    onAddExternalModel?: () => void;
}> = ({
    modelType,
    modelTypes,
    currentModel,
    models,
    loading,
    getAIModels,
    getAIModelsLoading,
    modelAvailable,
    handleAIModelTypeChange,
    handleAIModelChange,
    handleAIModelRemove,
    handleAIProviderChange,
    modelTypesDropdownItems,
    modelDropdownItems,
    disableNewChat,
    onClear,
    onAddExternalModel,
}) => {
    const dispatch = useAppDispatch();
    const [addExternalModel, setAddExternalModel] = useState(false);
    const [externalModelType, setExternalModel] = useState<string>(externalModelTypes[0].id);
    const [externalModelToken, setExternalModelToken] = useState<string>("");

    const handleAddExternalModel = useCallback(() => {
        setAddExternalModel(status => !status);
        onAddExternalModel?.();
    }, [onAddExternalModel]);

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

    const handleClear = useCallback(() => {
        onClear?.();
    }, [onClear]);

    const handleDeleteProvider = useCallback((id?: string) => {
        if (id) {
            dispatch(AIModelsActions.removeAIModelType({ id }));
        }
        dispatch(AIModelsActions.setCurrentModelType({ id: "" }));
        dispatch(AIModelsActions.setModels([]));
        dispatch(AIModelsActions.setCurrentModel(undefined));
    }, [dispatch]);

    return <div className="flex flex-col gap-4">
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
                <div className="flex items-center gap-sm self-end">
                    <Button
                        onClick={handleAddExternalModel}
                        data-testid="external-model-cancel"
                        variant="secondary"
                    >
                        <XMarkIcon className="w-4 h-4" /> Cancel
                    </Button>
                    <Button
                        onClick={handleExternalModelSubmit}
                        disabled={getAIModelsLoading}
                        data-testid="external-model-submit"
                    >
                        <CheckCircleIcon className="w-4 h-4" /> Submit
                    </Button>
                </div>
                <SheetFooter className="p-0">
                    <div className="text-xs text-neutral-500 mt-4 flex flex-col gap-2">
                        <div className="font-bold">Local Setup</div>
                        <div>
                            Go to <ExternalLink href="https://ollama.com/" className="font-semibold underline text-blue-600 hover:text-blue-800">Ollama</ExternalLink> and follow the installation instructions.
                        </div>
                        <div className="font-semibold">Downloading the Ollama Model</div>
                        <div>
                            Once installed, install the desired model you would like to use. In this guide, we will use <ExternalLink href="https://ollama.com/library/llama3.1" className="font-semibold underline text-blue-600 hover:text-blue-800">Llama3.1 8b</ExternalLink>. To install this model, run:
                        </div>
                        <div className="font-mono bg-neutral-100 dark:bg-neutral-900 rounded px-2 py-1 mb-1">
                            ollama run llama3.1
                        </div>
                        <div>
                            Check our documentation for more information on how to setup Ollama.
                        </div>
                        <Button variant="secondary" className="w-full mt-2" onClick={handleOpenDocs}>
                            Docs
                            <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                        </Button>
                    </div>
                </SheetFooter>
            </SheetContent>
        </Sheet>
        <div className="flex w-full justify-between">
            <div className="flex gap-2">
                <SearchSelect
                    options={modelTypesDropdownItems.map(item => ({
                        value: item.id,
                        label: item.label,
                        icon: item.icon,
                        rightIcon: item.extra?.isEnvironmentDefined ? <LockClosedIcon className="w-3 h-3 text-muted-foreground" /> : undefined,
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
                            <span className="flex items-center gap-sm text-green-500">
                                <PlusCircleIcon className="w-4 h-4 stroke-green-500" />
                                Add a provider
                            </span>
                        </CommandItem>
                    }
                />
                <SearchSelect
                    disabled={modelType == null}
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
            </div>
            <AlertDialog>
                <AlertDialogTrigger asChild>
                    <Button
                        data-testid="chat-delete-provider"
                        variant="secondary"
                        className={cn({
                            "hidden": disableNewChat,
                        })}
                    >
                        <TrashIcon className="w-4 h-4" /> Delete Provider
                    </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Provider</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete this provider? This action cannot be undone.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction asChild>
                            <Button
                                data-testid="chat-delete-provider-confirm"
                                onClick={() => handleDeleteProvider(modelType?.id)}
                                variant="destructive"
                            >
                                Delete
                            </Button>
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
        <div className={cn("flex items-center", {
            "hidden": disableNewChat,
        })}>
            <Button onClick={handleClear} disabled={loading} data-testid="chat-new-chat" variant="secondary">
                <ArrowPathIcon className="w-4 h-4" /> New Chat
            </Button>
        </div>
    </div>
}