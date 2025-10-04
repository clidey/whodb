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
import { AIProvidersActions, IAIProvider } from '../store/ai-providers';

export const useAIProviders = () => {
  const currentProviderId = useAppSelector(state => state.aiProviders.currentProviderId);
  const providers = useAppSelector(state => state.aiProviders.providers);
  const currentModel = useAppSelector(state => state.aiProviders.currentModel);
  const models = useAppSelector(state => state.aiProviders.models);
  const [modelAvailable, setModelAvailable] = useState(true);
  const [hasLoadedFromServer, setHasLoadedFromServer] = useState(false);

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

  // Load providers on mount and sync with server
  useEffect(() => {
    if (hasLoadedFromServer) {
      return;
    }

    getAiProviders({
      onCompleted(data) {
        const serverProviders = data.AIProviders || [];

        // Build deduplication map - use name+type as unique key
        const uniqueProviders = new Map<string, IAIProvider>();

        // First, add all server providers (they take priority)
        serverProviders.forEach(sp => {
          const key = `${sp.Name.toLowerCase().trim()}_${sp.Type.toLowerCase().trim()}`;

          // Try to find matching provider from localStorage to preserve API key
          const localMatch = providers.find(lp =>
            lp.name.toLowerCase().trim() === sp.Name.toLowerCase().trim() &&
            lp.type.toLowerCase().trim() === sp.Type.toLowerCase().trim()
          );

          uniqueProviders.set(key, {
            id: sp.Id,
            name: sp.Name,
            type: sp.Type,
            baseURL: sp.BaseURL || undefined,
            apiKey: sp.IsUserDefined && localMatch?.apiKey ? localMatch.apiKey : '',
            isEnvironmentDefined: sp.IsEnvironmentDefined,
            isUserDefined: sp.IsUserDefined,
            settings: sp.Settings ? JSON.parse(sp.Settings) : (localMatch?.settings || {}),
          });
        });

        // Then, add UI-only providers from localStorage (that don't exist on server)
        providers.forEach(p => {
          if (p.isUserDefined) {
            const key = `${p.name.toLowerCase().trim()}_${p.type.toLowerCase().trim()}`;
            // Only add if not already present (server providers take priority)
            if (!uniqueProviders.has(key)) {
              uniqueProviders.set(key, p);
            }
          }
        });

        // Convert map to array for final provider list
        const finalProviders = Array.from(uniqueProviders.values());

        dispatch(AIProvidersActions.setProviders(finalProviders));
        setHasLoadedFromServer(true);

        // Maintain current selection
        if (currentProviderId) {
          const stillExists = finalProviders.some(p => p.id === currentProviderId);
          if (!stillExists && finalProviders.length > 0) {
            dispatch(AIProvidersActions.setCurrentProvider({ id: finalProviders[0].id }));
          }
        } else if (!currentProviderId && finalProviders.length > 0) {
          dispatch(AIProvidersActions.setCurrentProvider({ id: finalProviders[0].id }));
        }
      },
    });
  }, [hasLoadedFromServer, getAiProviders, providers, currentProviderId, dispatch]);

  // Load models when provider changes
  useEffect(() => {
    if (currentProviderId && hasLoadedFromServer) {
      const provider = providers.find(p => p.id === currentProviderId);

      // For UI-added providers, ensure they exist on the backend before fetching models
      if (provider && provider.isUserDefined && provider.apiKey) {
        // First, try to update the provider on the backend to ensure it has the latest API key
        updateProvider({
          variables: {
            Id: provider.id,
            provider: {
              Name: provider.name,
              Type: provider.type,
              APIKey: provider.apiKey,
              BaseURL: provider.baseURL,
              Settings: provider.settings ? JSON.stringify(provider.settings) : undefined
            }
          }
        }).then(() => {
          // After ensuring provider exists on backend, fetch models
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
        }).catch(() => {
          // If update fails (provider doesn't exist), try to create it
          createProvider({
            variables: {
              provider: {
                Name: provider.name,
                Type: provider.type,
                APIKey: provider.apiKey,
                BaseURL: provider.baseURL,
                Settings: provider.settings ? JSON.stringify(provider.settings) : undefined
              }
            }
          }).then((result) => {
            if (result.data?.CreateAIProvider) {
              // Remove the old provider first
              dispatch(AIProvidersActions.removeProvider({ id: provider.id }));

              // Add the provider with the new backend ID
              const updatedProvider = {
                ...provider,
                id: result.data.CreateAIProvider.Id,
                apiKey: provider.apiKey  // Explicitly preserve the API key
              };

              // addProvider will now check for duplicates by name+type
              dispatch(AIProvidersActions.addProvider(updatedProvider));
              dispatch(AIProvidersActions.setCurrentProvider({ id: updatedProvider.id }));

              // Now fetch models with the new provider ID
              setModelAvailable(true);
              getAIModels({
                variables: { providerId: updatedProvider.id },
                onCompleted(data) {
                  dispatch(AIProvidersActions.setModels(data.AIModel || []));
                  if (data.AIModel && data.AIModel.length > 0) {
                    dispatch(AIProvidersActions.setCurrentModel(data.AIModel[0]));
                  }
                },
              });
            }
          }).catch((error) => {
            console.error('Failed to create provider on backend:', error);
            setModelAvailable(false);
            dispatch(AIProvidersActions.setModels([]));
            dispatch(AIProvidersActions.setCurrentModel(undefined));
          });
        });
      } else {
        // For environment-defined providers, just fetch models normally
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
    }
  }, [currentProviderId, hasLoadedFromServer, providers, dispatch, getAIModels, createProvider, updateProvider]);

  const handleProviderChange = useCallback((providerId: string) => {
    dispatch(AIProvidersActions.setCurrentProvider({ id: providerId }));
  }, [dispatch]);

  const handleModelChange = useCallback((model: string) => {
    dispatch(AIProvidersActions.setCurrentModel(model));
  }, [dispatch]);

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
          settings: data.UpdateAIProvider.Settings ? JSON.parse(data.UpdateAIProvider.Settings) : (settings || {})
        };
        dispatch(AIProvidersActions.updateProvider(updatedProvider));
        return updatedProvider;
      }
    } catch (error) {
      throw error;
    }
  }, [updateProvider, dispatch]);

  const handleCreateProvider = useCallback(async (
    name: string,
    type: string,
    apiKey?: string,
    baseURL?: string,
    settings?: Record<string, any>
  ) => {
    try {
      // Check if a provider with the same name and type already exists
      const key = `${name.toLowerCase()}_${type.toLowerCase()}`;
      const existingProvider = providers.find(p =>
        `${p.name.toLowerCase()}_${p.type.toLowerCase()}` === key
      );

      if (existingProvider) {
        // Update the existing provider instead of creating a duplicate
        return handleUpdateProvider(
          existingProvider.id,
          name,
          type,
          apiKey || '',
          baseURL,
          settings
        );
      }

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
          apiKey: apiKey || '',  // Store the API key from the input
          isEnvironmentDefined: data.CreateAIProvider.IsEnvironmentDefined,
          isUserDefined: data.CreateAIProvider.IsUserDefined,
          settings: data.CreateAIProvider.Settings ? JSON.parse(data.CreateAIProvider.Settings) : (settings || {})
        };
        dispatch(AIProvidersActions.addProvider(newProvider));
        dispatch(AIProvidersActions.setCurrentProvider({ id: newProvider.id }));
        return newProvider;
      }
    } catch (error) {
      throw error;
    }
  }, [createProvider, dispatch, providers, handleUpdateProvider]);

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
    getAiProviders({
      onCompleted(data) {
        const serverProviders = data.AIProviders || [];

        // Build deduplication map - use name+type as unique key
        const uniqueProviders = new Map<string, IAIProvider>();

        // First, add all server providers (they take priority)
        serverProviders.forEach(sp => {
          const key = `${sp.Name.toLowerCase().trim()}_${sp.Type.toLowerCase().trim()}`;

          // Try to find matching provider from localStorage to preserve API key
          const localMatch = providers.find(lp =>
            lp.name.toLowerCase().trim() === sp.Name.toLowerCase().trim() &&
            lp.type.toLowerCase().trim() === sp.Type.toLowerCase().trim()
          );

          uniqueProviders.set(key, {
            id: sp.Id,
            name: sp.Name,
            type: sp.Type,
            baseURL: sp.BaseURL || undefined,
            apiKey: sp.IsUserDefined && localMatch?.apiKey ? localMatch.apiKey : '',
            isEnvironmentDefined: sp.IsEnvironmentDefined,
            isUserDefined: sp.IsUserDefined,
            settings: sp.Settings ? JSON.parse(sp.Settings) : (localMatch?.settings || {}),
          });
        });

        // Then, add UI-only providers from localStorage (that don't exist on server)
        providers.forEach(p => {
          if (p.isUserDefined) {
            const key = `${p.name.toLowerCase().trim()}_${p.type.toLowerCase().trim()}`;
            // Only add if not already present (server providers take priority)
            if (!uniqueProviders.has(key)) {
              uniqueProviders.set(key, p);
            }
          }
        });

        // Convert map to array for final provider list
        const finalProviders = Array.from(uniqueProviders.values());

        dispatch(AIProvidersActions.setProviders(finalProviders));
      },
    });
  }, [getAiProviders, providers, dispatch]);

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