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

import { useLazyQuery } from "@apollo/client/react";
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
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Sheet,
    SheetContent,
    Separator,
    SheetFooter,
    toast
} from "@clidey/ux";
import { SearchSelect } from "./ux";
import { FC, ReactElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { GetAiModelsDocument, GetAiProvidersDocument } from "@graphql";
import { AIModelsActions, availableExternalModelTypes, type IAIModelType } from "../store/ai-models";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import { ensureModelsArray, ensureModelTypesArray } from "../utils/ai-models-helper";
import { ExternalLink } from "../utils/external-links";
import { v4 as uuidv4 } from 'uuid';
import { useTranslation } from "../hooks/use-translation";
import { getAIProviderOverrides } from "../config/ai-provider-registry";
import { persistAISelection } from "../config/ai-persistence";
import {
    ArrowPathIcon,
    ArrowTopRightOnSquareIcon,
    CheckCircleIcon,
    ChevronDownIcon,
    ExclamationCircleIcon,
    LockClosedIcon,
    PlusCircleIcon,
    SparklesIcon,
    TrashIcon,
    XMarkIcon
} from "./heroicons";
import { Icons } from "./icons";

export const externalModelTypes = availableExternalModelTypes.map((model) => ({
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
    const [unavailableProviders, setUnavailableProviders] = useState<Set<string>>(new Set());

    // Get persisted AI selection from platform store (EE only)
    const platformState = useAppSelector(state => (state as any).platform);
    const persistedProviderId = platformState?.selectedAIProviderId;
    const persistedModel = platformState?.selectedAIModel;

    const dispatch = useAppDispatch();

    const [getAiProviders, { loading }] = useLazyQuery(GetAiProvidersDocument);
    const [getAIModels, { loading: getAIModelsLoading }] = useLazyQuery(GetAiModelsDocument, {
        fetchPolicy: "network-only",
    });

    const markProviderUnavailable = useCallback((providerId: string) => {
        setUnavailableProviders(prev => {
            if (prev.has(providerId)) return prev;
            const next = new Set(prev);
            next.add(providerId);
            return next;
        });
    }, []);

    const markProviderAvailable = useCallback((providerId: string) => {
        setUnavailableProviders(prev => {
            if (!prev.has(providerId)) return prev;
            const next = new Set(prev);
            next.delete(providerId);
            return next;
        });
    }, []);

    const handleAIModelsError = useCallback(() => {
        setModelAvailable(false);
        dispatch(AIModelsActions.setModels([]));
        dispatch(AIModelsActions.setCurrentModel(undefined));
    }, [dispatch]);

    const fetchAIModels = useCallback(async (variables: {
        providerId?: string;
        modelType: string;
        token?: string;
    }) => {
        const { data, error } = await getAIModels({ variables });

        if (error) {
            throw error;
        }

        return data?.AIModel ?? [];
    }, [getAIModels]);

    const handleAIModelTypeChange = useCallback((item: string) => {
        const modelType = modelTypes.find(model => model.id === item);
        if (modelType == null) {
            return;
        }
        setModelAvailable(true);
        void fetchAIModels({
                providerId: modelType.id,
                modelType: modelType.modelType,
                token: modelType.token,
            })
            .then((aiModels) => {
                dispatch(AIModelsActions.setModels(aiModels));
                if (aiModels.length > 0) {
                    dispatch(AIModelsActions.setCurrentModel(aiModels[0]));
                    markProviderAvailable(modelType.id);
                } else {
                    markProviderUnavailable(modelType.id);
                }
            })
            .catch(() => {
                markProviderUnavailable(modelType.id);
                handleAIModelsError();
            });
    }, [dispatch, fetchAIModels, handleAIModelsError, modelTypes]);

    const handleAIModelChange = useCallback((item: string) => {
        dispatch(AIModelsActions.setCurrentModel(item));

        if (modelType) {
            persistAISelection({ providerId: modelType.id, model: item });
        }
    }, [dispatch, modelType]);

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

        persistAISelection({ providerId: item, model: null });
    }, [handleAIModelTypeChange, dispatch]);

    const retryProvider = useCallback((providerId: string) => {
        handleAIModelTypeChange(providerId);
    }, [handleAIModelTypeChange]);

    const stateRef = useRef({ modelType, currentModel, modelTypes });
    stateRef.current = { modelType, currentModel, modelTypes };

    const loadProviders = useCallback(async () => {
        const state = stateRef.current;
        const { modelType: currentType, currentModel: savedModel, modelTypes: currentModelTypes } = state;
        const overrides = getAIProviderOverrides();
        const isPlatformMode = overrides?.isActive() ?? false;

        // Set selection immediately from platform store if available (before async loading)
        if (persistedProviderId && !currentType) {
            dispatch(AIModelsActions.setCurrentModelType({ id: persistedProviderId }));
            if (persistedModel) {
                dispatch(AIModelsActions.setCurrentModel(persistedModel));
            }
        }

        const userAddedProviders = isPlatformMode
            ? []
            : currentModelTypes.filter(model => model.token != null && model.token !== "");

        const [envResult, platformProviders] = await Promise.all([
            getAiProviders().catch(() => ({ data: undefined, error: undefined })),
            isPlatformMode && overrides ? overrides.loadProviders() : Promise.resolve([]),
        ]);

        const aiProviders = envResult.data?.AIProviders || [];

        const initialModelTypes = userAddedProviders.filter(model =>
            model.token != null && model.token !== ""
        );

        const newProviders = aiProviders.filter(provider =>
            !initialModelTypes.some(model => model.id === provider.ProviderId)
        );

        const finalModelTypes: IAIModelType[] = [
            ...newProviders.map(provider => ({
                id: provider.ProviderId,
                modelType: provider.Type,
                name: provider.Name,
                isEnvironmentDefined: provider.IsEnvironmentDefined,
                isGeneric: provider.IsGeneric,
            })),
            ...platformProviders,
            ...initialModelTypes,
        ];

        const waitingForPlatform = currentType?.isPlatformProvider && !isPlatformMode;

        // Check if we have a persisted provider ID to restore from platform store
        const shouldRestoreSelection = !currentType && persistedProviderId && finalModelTypes.some(m => m.id === persistedProviderId);

        if (currentType && !finalModelTypes.some(model => model.id === currentType.id) && !waitingForPlatform && !shouldRestoreSelection) {
            dispatch(AIModelsActions.setCurrentModelType({ id: "" }));
            dispatch(AIModelsActions.setModels([]));
            dispatch(AIModelsActions.setCurrentModel(undefined));
        }

        dispatch(AIModelsActions.setModelTypes(finalModelTypes));

        if (waitingForPlatform) {
            return;
        }

        // Restore selection if available
        let selectedProvider = currentType && finalModelTypes.some(m => m.id === currentType.id)
            ? currentType
            : null;

        if (!selectedProvider && shouldRestoreSelection) {
            selectedProvider = finalModelTypes.find(m => m.id === persistedProviderId) || null;
            if (selectedProvider) {
                dispatch(AIModelsActions.setCurrentModelType({ id: selectedProvider.id }));
            }
        }

        if (!selectedProvider && finalModelTypes.length > 0) {
            const firstProvider = finalModelTypes[0];
            dispatch(AIModelsActions.setCurrentModelType({ id: firstProvider.id }));

            void fetchAIModels({
                    providerId: firstProvider.id,
                    modelType: firstProvider.modelType ?? "",
                    token: firstProvider.token ?? "",
                })
                .then((aiModels) => {
                    dispatch(AIModelsActions.setModels(aiModels));
                    if (aiModels.length > 0) {
                        dispatch(AIModelsActions.setCurrentModel(aiModels[0]));
                        markProviderAvailable(firstProvider.id);
                    } else {
                        markProviderUnavailable(firstProvider.id);
                    }
                })
                .catch(() => {
                    markProviderUnavailable(firstProvider.id);
                    handleAIModelsError();
                });
        } else if (selectedProvider) {
            dispatch(AIModelsActions.setCurrentModelType({ id: selectedProvider.id }));
            void fetchAIModels({
                    providerId: selectedProvider.id,
                    modelType: selectedProvider.modelType ?? "",
                    token: selectedProvider.token ?? "",
                })
                .then((aiModels) => {
                    dispatch(AIModelsActions.setModels(aiModels));
                    if (aiModels.length > 0) {
                        // Try to restore persisted model from platform store, fall back to saved model, then first model
                        const modelToSelect = (persistedModel && aiModels.includes(persistedModel))
                            ? persistedModel
                            : (savedModel && aiModels.includes(savedModel))
                                ? savedModel
                                : aiModels[0];
                        dispatch(AIModelsActions.setCurrentModel(modelToSelect));
                        markProviderAvailable(selectedProvider.id);
                    } else {
                        markProviderUnavailable(selectedProvider.id);
                    }
                })
                .catch(() => {
                    markProviderUnavailable(selectedProvider.id);
                    handleAIModelsError();
                });
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    useEffect(() => {
        void loadProviders();

        const overrides = getAIProviderOverrides();
        if (overrides?.onActivate) {
            return overrides.onActivate(() => { void loadProviders(); });
        }
    }, [loadProviders]);

    const modelTypesDropdownItems = useMemo(() => {
        return modelTypes.filter(modelType => modelType != null && modelType.modelType != null).map(modelType => ({
            id: modelType.id,
            label: modelType.name || modelType.modelType,
            icon: modelType.isGeneric
                ? <SparklesIcon className="w-4 h-4" data-testid="generic-sparkles-icon" />
                : (Icons.Logos as Record<string, ReactElement>)[modelType.modelType.replace("-", "")],
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
        unavailableProviders,
        handleAIModelsError,
        handleAIModelTypeChange,
        handleAIModelChange,
        handleAIModelRemove,
        handleAIProviderChange,
        markProviderAvailable,
        retryProvider,
        modelTypesDropdownItems,
        modelDropdownItems,
    }
}

export const AIProvider: FC<ReturnType<typeof useAI> & {
    disableNewChat?: boolean;
    disableClear?: boolean;
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
    unavailableProviders,
    retryProvider,
    handleAIModelsError,
    handleAIModelTypeChange,
    handleAIModelChange,
    handleAIModelRemove,
    handleAIProviderChange,
    modelTypesDropdownItems,
    modelDropdownItems,
    disableNewChat,
    disableClear,
    onClear,
    onAddExternalModel,
}) => {
    const { t } = useTranslation('components/ai');
    const newUIEnabled = useAppSelector(state => state.settings.newUIEnabled);
    const dispatch = useAppDispatch();
    const [addExternalModel, setAddExternalModel] = useState(false);
    const [externalModelType, setExternalModel] = useState<string>(externalModelTypes[0].id);
    const [externalModelToken, setExternalModelToken] = useState<string>("");
    const [externalModelName, setExternalModelName] = useState<string>("");

    const handleAddExternalModel = useCallback(() => {
        setAddExternalModel(status => !status);
        onAddExternalModel?.();
    }, [onAddExternalModel]);

    const handleExternalModelChange = useCallback((item: string) => {
        setExternalModel(item);
    }, []);

    const handleExternalModelSubmit = useCallback(() => {
        // Validate token is provided
        if (!externalModelToken || externalModelToken.trim().length === 0) {
            toast.error(t('tokenRequired'));
            return;
        }

        dispatch(AIModelsActions.setCurrentModel(undefined));
        dispatch(AIModelsActions.setModels([]));
        const overrides = getAIProviderOverrides();
        const isPlatformMode = overrides?.isActive() ?? false;

        void getAIModels({
            variables: {
                modelType: externalModelType,
                token: externalModelToken,
            }
        }).then(async ({ data, error }) => {
            if (error) {
                throw error;
            }

            const aiModels = data?.AIModel ?? [];
            dispatch(AIModelsActions.setModels(aiModels));

            let id: string;
            if (isPlatformMode && overrides) {
                const result = await overrides.addProvider({
                    modelType: externalModelType,
                    name: externalModelName || externalModelType,
                    token: externalModelToken,
                });
                if (!result) throw new Error('Failed to create provider');
                id = result.id;
                dispatch(AIModelsActions.addAIModelType({
                    id,
                    modelType: externalModelType,
                    name: externalModelName || externalModelType,
                    isPlatformProvider: true,
                }));
            } else {
                id = uuidv4();
                dispatch(AIModelsActions.addAIModelType({
                    id,
                    modelType: externalModelType,
                    name: externalModelName || externalModelType,
                    token: externalModelToken,
                }));
            }

            dispatch(AIModelsActions.setCurrentModelType({ id }));
            setExternalModel(externalModelTypes[0].id);
            setExternalModelToken("");
            setExternalModelName("");
            setAddExternalModel(false);
            if (aiModels.length > 0) {
                dispatch(AIModelsActions.setCurrentModel(aiModels[0]));
            }
        }).catch((error: unknown) => {
            handleAIModelsError();
            const errorMessage = error instanceof Error
                ? error.message
                : String((error as { message?: string }).message ?? t('unknownError'));
            toast.error(`${t('unableToConnect')}: ${errorMessage}`);
        });
    }, [dispatch, externalModelName, externalModelToken, externalModelType, getAIModels, handleAIModelsError, t]);

    const handleOpenDocs = useCallback(() => {
        window.open("https://docs.whodb.com/ai/introduction", "_blank");
    }, []);

    const handleClear = useCallback(() => {
        onClear?.();
    }, [onClear]);

    const handleDeleteProvider = useCallback(async (id?: string) => {
        if (id) {
            const overrides = getAIProviderOverrides();
            if (overrides?.isActive()) {
                await overrides.deleteProvider(id).catch(() => {});
            }
            dispatch(AIModelsActions.removeAIModelType({ id }));
        }
        dispatch(AIModelsActions.setCurrentModelType({ id: "" }));
        dispatch(AIModelsActions.setModels([]));
        dispatch(AIModelsActions.setCurrentModel(undefined));
    }, [dispatch]);

    return <div className="flex flex-col gap-4" data-testid="ai-provider">
        <Sheet open={addExternalModel} onOpenChange={setAddExternalModel}>
            <SheetContent className={cn("max-w-md mx-auto w-full flex flex-col gap-4", {
                "px-8 py-10": !newUIEnabled,
            })}>
                <div className="flex flex-col gap-4">
                    <div className="text-lg font-semibold mb-2">{t('addExternalModel')}</div>
                    <div className="flex flex-col gap-2">
                        <Label>{t('modelType')}</Label>
                        <Select
                            value={externalModelType}
                            onValueChange={handleExternalModelChange}
                        >
                            <SelectTrigger className="w-full" data-testid="external-model-type-select">
                                <SelectValue placeholder={t('selectProvider')} />
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
                        <Label>{t('name')}</Label>
                        <Input
                            value={externalModelName ?? ""}
                            onChange={e => setExternalModelName(e.target.value)}
                            placeholder={externalModelType}
                        />
                    </div>
                    <div className="flex flex-col gap-2">
                        <Label>{t('token')}</Label>
                        <Input
                            value={externalModelToken ?? ""}
                            onChange={e => setExternalModelToken(e.target.value)}
                            type="password"
                        />
                    </div>
                </div>
                <div className={cn("flex items-center gap-sm self-end", {
                    "mt-4": newUIEnabled,
                })}>
                    <Button
                        onClick={handleAddExternalModel}
                        data-testid="external-model-cancel"
                        variant="secondary"
                    >
                        <XMarkIcon className="w-4 h-4" /> {t('cancel')}
                    </Button>
                    <Button
                        onClick={handleExternalModelSubmit}
                        disabled={getAIModelsLoading}
                        data-testid="external-model-submit"
                    >
                        <CheckCircleIcon className="w-4 h-4" /> {t('submit')}
                    </Button>
                </div>
                <SheetFooter className="p-0">
                    <div className="text-xs text-neutral-500 mt-4 flex flex-col gap-2">
                        <div className="font-bold">{t('localSetup')}</div>
                        <div>
                            {t('ollamaSetupText').split('<0>')[0]}
                            <ExternalLink href="https://ollama.com/" className="font-semibold underline text-blue-600 hover:text-blue-800">Ollama</ExternalLink>
                            {t('ollamaSetupText').split('</0>')[1]}
                        </div>
                        <div className="font-semibold">{t('downloadingModel')}</div>
                        <div>
                            {t('ollamaDownloadText').split('<0>')[0]}
                            <ExternalLink href="https://ollama.com/library/llama3.1" className="font-semibold underline text-blue-600 hover:text-blue-800">Llama3.1 8b</ExternalLink>
                            {t('ollamaDownloadText').split('</0>')[1]}
                        </div>
                        <div className="font-mono bg-neutral-100 dark:bg-neutral-900 rounded px-2 py-1 mb-1">
                            {t('ollamaRunCommand')}
                        </div>
                        <div>
                            {t('ollamaDocsText')}
                        </div>
                        <Button variant="secondary" className="w-full mt-2" onClick={handleOpenDocs}>
                            {t('docs')}
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
                        rightIcon: <span className="flex items-center gap-1">
                            {unavailableProviders.has(item.id) && <span title={t('providerUnavailable')}><ExclamationCircleIcon className="w-3 h-3 text-amber-500" /></span>}
                            {item.extra?.isEnvironmentDefined && <span title={t('environmentDefined')}><LockClosedIcon className="w-3 h-3 text-muted-foreground" /></span>}
                        </span>,
                    }))}
                    value={modelType?.id}
                    onChange={id => {
                        const item = modelTypesDropdownItems.find(i => i.id === id);
                        if (item) handleAIProviderChange(item.id);
                    }}
                    placeholder={t('selectProvider')}
                    side="right"
                    align="start"
                    extraOptions={
                        newUIEnabled ? <>
                            <Separator />
                            <CommandItem
                                key="__add__"
                                value="__add__"
                                onSelect={handleAddExternalModel}
                            >
                                <span className="mr-2 h-4 w-4 shrink-0" />
                                <PlusCircleIcon className="w-4 h-4" />
                                {t('addProvider')}
                            </CommandItem>
                        </> : (<>
                            <CommandItem
                                key="__add__"
                                value="__add__"
                                onSelect={handleAddExternalModel}
                            >
                                <span className="flex items-center gap-sm text-green-500">
                                    <PlusCircleIcon className="w-4 h-4 stroke-green-500" />
                                    {t('addProvider')}
                                </span>
                            </CommandItem></>
                        )
                    }
                    rightIcon={<ChevronDownIcon className="w-4 h-4" />}
                    buttonProps={{
                        "data-testid": "ai-provider-select",
                    }}
                />
                {modelType && unavailableProviders.has(modelType.id) && (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => retryProvider(modelType.id)}
                        title={t('providerUnavailable')}
                        className="px-2"
                    >
                        <ArrowPathIcon className="w-4 h-4 text-amber-500" />
                    </Button>
                )}
                <SearchSelect
                    disabled={modelType == null || getAIModelsLoading}
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
                    placeholder={t('selectModel')}
                    side="right"
                    align="start"
                    rightIcon={<ChevronDownIcon className="w-4 h-4" />}
                    buttonProps={{
                        "data-testid": "ai-model-select",
                    }}
                />
            </div>
            <AlertDialog>
                <AlertDialogTrigger asChild>
                    <Button
                        data-testid="chat-delete-provider"
                        variant="secondary"
                        className={cn({
                            "hidden": disableNewChat || modelType?.isEnvironmentDefined || modelType?.isPlatformProvider,
                        })}
                    >
                        <TrashIcon className="w-4 h-4" /> {t('deleteProvider')}
                    </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('deleteProvider')}</AlertDialogTitle>
                        <AlertDialogDescription>
                            {t('deleteProviderConfirm')}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>{t('cancel')}</AlertDialogCancel>
                        <AlertDialogAction asChild>
                            <Button
                                data-testid="chat-delete-provider-confirm"
                                onClick={() => handleDeleteProvider(modelType?.id)}
                                variant="destructive"
                            >
                                {t('delete')}
                            </Button>
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
        <div className={cn("flex items-center", {
            "hidden": disableNewChat,
        })}>
            <Button onClick={handleClear} disabled={loading || disableClear} data-testid="chat-new-chat" variant="secondary">
                <ArrowPathIcon className="w-4 h-4" /> {t('newChat')}
            </Button>
        </div>
    </div>
}
