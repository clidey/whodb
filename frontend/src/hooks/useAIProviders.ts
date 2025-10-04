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

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  useGetAiProvidersLazyQuery,
  useGetAiModelsLazyQuery,
  useCreateAiProviderMutation,
  useUpdateAiProviderMutation,
  useDeleteAiProviderMutation
} from '../generated/graphql';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import { AIProvidersActions } from '../store/ai-providers';

export const useAIProviders = () => {
  const currentProviderId = useAppSelector(state => state.aiProviders.currentProviderId);
  const providers = useAppSelector(state => state.aiProviders.providers);
  const currentModel = useAppSelector(state => state.aiProviders.currentModel);
  const models = useAppSelector(state => state.aiProviders.models);
  const [modelAvailable, setModelAvailable] = useState(true);

  const dispatch = useAppDispatch();

  const [getAiProviders, { loading: loadingProviders }] = useGetAiProvidersLazyQuery({
    fetchPolicy: 'network-only',
  });

  const [getAIModels, { loading: loadingModels }] = useGetAiModelsLazyQuery({
    onError() {
      setModelAvailable(false);
      dispatch(AIProvidersActions.setModels([]));
      dispatch(AIProvidersActions.setCurrentModel(undefined));
    },
    fetchPolicy: 'network-only',
  });

  const [createProvider] = useCreateAiProviderMutation();
  const [updateProvider] = useUpdateAiProviderMutation();
  const [deleteProvider] = useDeleteAiProviderMutation();

  // Get current provider object
  const currentProvider = useMemo(() => {
    return providers.find(p => p.id === currentProviderId);
  }, [providers, currentProviderId]);

  // Load providers on mount
  useEffect(() => {
    getAiProviders({
      onCompleted(data) {
        const serverProviders = data.AIProviders || [];

        // Map server providers and preserve API keys from localStorage
        const finalProviders = serverProviders.map(sp => {
          const localMatch = providers.find(lp => lp.id === sp.Id);
          return {
            id: sp.Id,
            name: sp.Name,
            type: sp.Type,
            baseURL: sp.BaseURL || undefined,
            apiKey: sp.IsUserDefined && localMatch?.apiKey ? localMatch.apiKey : '',
            isEnvironmentDefined: sp.IsEnvironmentDefined,
            isUserDefined: sp.IsUserDefined,
            settings: sp.Settings ? JSON.parse(sp.Settings) : (localMatch?.settings || {}),
          };
        });

        dispatch(AIProvidersActions.setProviders(finalProviders));

        // Maintain current selection
        if (currentProviderId && !finalProviders.some(p => p.id === currentProviderId)) {
          if (finalProviders.length > 0) {
            dispatch(AIProvidersActions.setCurrentProvider({ id: finalProviders[0].id }));
          }
        } else if (!currentProviderId && finalProviders.length > 0) {
          dispatch(AIProvidersActions.setCurrentProvider({ id: finalProviders[0].id }));
        }
      },
    });
  }, []); // Only run on mount

  // Load models when provider changes
  useEffect(() => {
    if (currentProviderId) {
      setModelAvailable(true);
      getAIModels({
        variables: { providerId: currentProviderId },
        onCompleted(data) {
          dispatch(AIProvidersActions.setModels(data.AIModel || []));
          if (data.AIModel && data.AIModel.length > 0) {
            dispatch(AIProvidersActions.setCurrentModel(data.AIModel[0]));
          }
        },
      });
    }
  }, [currentProviderId, dispatch, getAIModels]);

  const handleProviderChange = useCallback((providerId: string) => {
    dispatch(AIProvidersActions.setCurrentProvider({ id: providerId }));
  }, [dispatch]);

  const handleModelChange = useCallback((model: string) => {
    dispatch(AIProvidersActions.setCurrentModel(model));
  }, [dispatch]);

  const handleCreateProvider = useCallback(async (
    name: string,
    type: string,
    apiKey?: string,
    baseURL?: string,
    settings?: Record<string, any>
  ) => {
    try {
      const { data } = await createProvider({
        variables: {
          provider: {
            Name: name,
            Type: type,
            APIKey: apiKey,
            BaseURL: baseURL,
            Settings: settings ? JSON.stringify(settings) : undefined
          }
        }
      });

      if (data?.CreateAIProvider) {
        const newProvider = {
          id: data.CreateAIProvider.Id,
          name: data.CreateAIProvider.Name,
          type: data.CreateAIProvider.Type,
          baseURL: data.CreateAIProvider.BaseURL || undefined,
          apiKey: apiKey || '',
          isEnvironmentDefined: data.CreateAIProvider.IsEnvironmentDefined,
          isUserDefined: data.CreateAIProvider.IsUserDefined,
          settings: data.CreateAIProvider.Settings ? JSON.parse(data.CreateAIProvider.Settings) : {}
        };
        dispatch(AIProvidersActions.addProvider(newProvider));
        dispatch(AIProvidersActions.setCurrentProvider({ id: newProvider.id }));
        return newProvider;
      }
    } catch (error) {
      throw error;
    }
  }, [createProvider, dispatch]);

  const handleUpdateProvider = useCallback(async (
    id: string,
    name: string,
    type: string,
    apiKey: string,
    baseURL?: string,
    settings?: Record<string, any>
  ) => {
    try {
      const { data } = await updateProvider({
        variables: {
          Id: id,
          provider: {
            Name: name,
            Type: type,
            APIKey: apiKey,
            BaseURL: baseURL,
            Settings: settings ? JSON.stringify(settings) : undefined
          }
        }
      });

      if (data?.UpdateAIProvider) {
        const updatedProvider = {
          id: data.UpdateAIProvider.Id,
          name: data.UpdateAIProvider.Name,
          type: data.UpdateAIProvider.Type,
          baseURL: data.UpdateAIProvider.BaseURL || undefined,
          apiKey: apiKey || '', // Always use the provided apiKey
          isEnvironmentDefined: data.UpdateAIProvider.IsEnvironmentDefined,
          isUserDefined: data.UpdateAIProvider.IsUserDefined,
          settings: data.UpdateAIProvider.Settings ? JSON.parse(data.UpdateAIProvider.Settings) : {}
        };
        dispatch(AIProvidersActions.updateProvider(updatedProvider));
        return updatedProvider;
      }
    } catch (error) {
      throw error;
    }
  }, [updateProvider, dispatch]);

  const handleDeleteProvider = useCallback(async (id: string) => {
    try {
      const { data } = await deleteProvider({
        variables: { Id: id }
      });

      if (data?.DeleteAIProvider?.Status) {
        dispatch(AIProvidersActions.removeProvider({ id }));

        // If this was the current provider, select another one
        if (currentProviderId === id) {
          const remainingProviders = providers.filter(p => p.id !== id);
          if (remainingProviders.length > 0) {
            dispatch(AIProvidersActions.setCurrentProvider({ id: remainingProviders[0].id }));
          }
        }
        return true;
      }
      return false;
    } catch (error) {
      throw error;
    }
  }, [deleteProvider, dispatch, currentProviderId, providers]);

  const refreshProviders = useCallback(() => {
    getAiProviders();
  }, [getAiProviders]);

  return {
    currentProviderId,
    currentProvider,
    providers,
    currentModel,
    models,
    modelAvailable,
    loadingProviders,
    loadingModels,
    handleProviderChange,
    handleModelChange,
    handleCreateProvider,
    handleUpdateProvider,
    handleDeleteProvider,
    refreshProviders,
  };
};