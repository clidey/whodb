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

import { FC, ReactElement, useCallback, useState } from 'react';
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
import {
  ArrowPathIcon,
  CheckCircleIcon,
  CogIcon,
  LockClosedIcon,
  PlusIcon,
  TrashIcon,
  XMarkIcon
} from "./heroicons";
import { useAIProviders } from '../hooks/useAIProviders';
import { Icons } from './icons';

interface AIProviderSelectProps {
  disableNewChat?: boolean;
  onClear?: () => void;
}

export const AIProviderSelect: FC<AIProviderSelectProps> = ({
  disableNewChat,
  onClear,
}) => {
  const {
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
  } = useAIProviders();

  const [addProviderOpen, setAddProviderOpen] = useState(false);
  const [providerName, setProviderName] = useState('');
  const [providerType, setProviderType] = useState('Ollama');
  const [providerApiKey, setProviderApiKey] = useState('');
  const [providerBaseURL, setProviderBaseURL] = useState('');
  const [providerSettings, setProviderSettings] = useState<Record<string, any>>({
    temperature: 0.7,
    max_tokens: 2048
  });

  const [settingsOpen, setSettingsOpen] = useState(false);
  const [editingSettings, setEditingSettings] = useState<Record<string, any>>({});

  const providerTypes = [
    { id: 'Ollama', label: 'Ollama', icon: (Icons.Logos as Record<string, ReactElement>)['Ollama'] },
    { id: 'ChatGPT', label: 'ChatGPT', icon: (Icons.Logos as Record<string, ReactElement>)['ChatGPT'] },
    { id: 'Anthropic', label: 'Anthropic', icon: (Icons.Logos as Record<string, ReactElement>)['Anthropic'] },
    { id: 'OpenAI-Compatible', label: 'OpenAI Compatible', icon: (Icons.Logos as Record<string, ReactElement>)['ChatGPT'] },
  ];

  const handleSubmitProvider = useCallback(async () => {
    try {
      await handleCreateProvider(
        providerName,
        providerType,
        providerApiKey || undefined,
        providerBaseURL || undefined,
        providerSettings
      );

      // Reset form
      setProviderName('');
      setProviderType('Ollama');
      setProviderApiKey('');
      setProviderBaseURL('');
      setProviderSettings({ temperature: 0.7, max_tokens: 2048 });
      setAddProviderOpen(false);

      toast.success('Provider added successfully');
    } catch (error: any) {
      toast.error(`Failed to add provider: ${error.message}`);
    }
  }, [handleCreateProvider, providerName, providerType, providerApiKey, providerBaseURL, providerSettings]);

  const providerDropdownItems = providers.map(provider => ({
    value: provider.id,
    label: provider.name,
    icon: (Icons.Logos as Record<string, ReactElement>)[provider.type.replace("-", "")] || undefined,
    rightIcon: provider.isEnvironmentDefined ? <LockClosedIcon className="w-3 h-3 text-muted-foreground" /> : undefined,
  }));

  const modelDropdownItems = models.map(model => ({
    value: model,
    label: model,
  }));

  const handleClear = useCallback(() => {
    onClear?.();
  }, [onClear]);

  const handleOpenSettings = useCallback(() => {
    if (currentProvider) {
      setEditingSettings(currentProvider.settings || { temperature: 0.7, max_tokens: 2048 });
      setSettingsOpen(true);
    }
  }, [currentProvider]);

  const handleSaveSettings = useCallback(async () => {
    if (!currentProvider || currentProvider.isEnvironmentDefined) return;

    try {
      await handleUpdateProvider(
        currentProvider.id,
        currentProvider.name,
        currentProvider.type,
        currentProvider.apiKey,
        currentProvider.baseURL,
        editingSettings
      );
      setSettingsOpen(false);
      toast.success('Settings updated successfully');
    } catch (error: any) {
      toast.error(`Failed to update settings: ${error.message}`);
    }
  }, [currentProvider, editingSettings, handleUpdateProvider]);

  return (
    <div className="flex flex-col gap-4">
      <Sheet open={addProviderOpen} onOpenChange={setAddProviderOpen}>
        <SheetContent className="max-w-md mx-auto w-full px-8 py-10 flex flex-col gap-4 overflow-y-auto">
          <div className="flex flex-col gap-4">
            <div className="text-lg font-semibold mb-2">Add AI Provider</div>

            <div className="flex flex-col gap-2">
              <Label>Provider Name</Label>
              <Input
                value={providerName}
                onChange={e => setProviderName(e.target.value)}
                placeholder="e.g., My ChatGPT Provider"
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label>Provider Type</Label>
              <Select value={providerType} onValueChange={setProviderType}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select Provider Type" />
                </SelectTrigger>
                <SelectContent>
                  {providerTypes.map(item => (
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

            {providerType !== 'Ollama' && (
              <div className="flex flex-col gap-2">
                <Label>API Key</Label>
                <Input
                  value={providerApiKey}
                  onChange={e => setProviderApiKey(e.target.value)}
                  type="password"
                  placeholder="Enter your API key"
                />
              </div>
            )}

            <div className="flex flex-col gap-2">
              <Label>Base URL (Optional)</Label>
              <Input
                value={providerBaseURL}
                onChange={e => setProviderBaseURL(e.target.value)}
                placeholder={
                  providerType === 'Ollama' ? 'http://localhost:11434' :
                  providerType === 'ChatGPT' ? 'https://api.openai.com/v1' :
                  providerType === 'Anthropic' ? 'https://api.anthropic.com/v1' :
                  'https://your-api-endpoint.com'
                }
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label>Temperature</Label>
              <Input
                type="number"
                min="0"
                max="2"
                step="0.1"
                value={providerSettings.temperature}
                onChange={e => setProviderSettings(prev => ({
                  ...prev,
                  temperature: parseFloat(e.target.value)
                }))}
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label>Max Tokens</Label>
              <Input
                type="number"
                min="1"
                max="32000"
                value={providerSettings.max_tokens}
                onChange={e => setProviderSettings(prev => ({
                  ...prev,
                  max_tokens: parseInt(e.target.value)
                }))}
              />
            </div>
          </div>

          <div className="flex items-center gap-2 self-end">
            <Button
              onClick={() => setAddProviderOpen(false)}
              variant="secondary"
            >
              <XMarkIcon className="w-4 h-4" /> Cancel
            </Button>
            <Button
              onClick={handleSubmitProvider}
              disabled={!providerName || (providerType !== 'Ollama' && !providerApiKey)}
            >
              <CheckCircleIcon className="w-4 h-4" /> Add Provider
            </Button>
          </div>
        </SheetContent>
      </Sheet>

      <Sheet open={settingsOpen} onOpenChange={setSettingsOpen}>
        <SheetContent className="max-w-md mx-auto w-full px-8 py-10 flex flex-col gap-4 overflow-y-auto">
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-2">
              <span className="text-lg font-semibold">Provider Settings</span>
              {currentProvider?.isEnvironmentDefined && (
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <LockClosedIcon className="w-3 h-3" />
                  Read-only
                </span>
              )}
            </div>

            {currentProvider && (
              <>
                <div className="flex flex-col gap-2">
                  <Label>Provider Name</Label>
                  <Input
                    value={currentProvider.name}
                    disabled
                    className="opacity-60"
                  />
                </div>

                <div className="flex flex-col gap-2">
                  <Label>Provider Type</Label>
                  <Input
                    value={currentProvider.type}
                    disabled
                    className="opacity-60"
                  />
                </div>

                {currentProvider.baseURL && (
                  <div className="flex flex-col gap-2">
                    <Label>Base URL</Label>
                    <Input
                      value={currentProvider.baseURL}
                      disabled
                      className="opacity-60"
                    />
                  </div>
                )}

                <div className="flex flex-col gap-2">
                  <Label>Temperature</Label>
                  <Input
                    type="number"
                    min="0"
                    max="2"
                    step="0.1"
                    value={editingSettings.temperature || 0.7}
                    onChange={e => setEditingSettings(prev => ({
                      ...prev,
                      temperature: parseFloat(e.target.value)
                    }))}
                    disabled={currentProvider.isEnvironmentDefined}
                    className={currentProvider.isEnvironmentDefined ? "opacity-60" : ""}
                  />
                  <span className="text-xs text-muted-foreground">
                    Controls randomness: 0 = deterministic, 2 = very random
                  </span>
                </div>

                <div className="flex flex-col gap-2">
                  <Label>Max Tokens</Label>
                  <Input
                    type="number"
                    min="1"
                    max="32000"
                    value={editingSettings.max_tokens || 2048}
                    onChange={e => setEditingSettings(prev => ({
                      ...prev,
                      max_tokens: parseInt(e.target.value)
                    }))}
                    disabled={currentProvider.isEnvironmentDefined}
                    className={currentProvider.isEnvironmentDefined ? "opacity-60" : ""}
                  />
                  <span className="text-xs text-muted-foreground">
                    Maximum number of tokens to generate in response
                  </span>
                </div>
              </>
            )}
          </div>

          <div className="flex items-center gap-2 self-end">
            <Button
              onClick={() => setSettingsOpen(false)}
              variant="secondary"
            >
              <XMarkIcon className="w-4 h-4" />
              {currentProvider?.isEnvironmentDefined ? 'Close' : 'Cancel'}
            </Button>
            {currentProvider && !currentProvider.isEnvironmentDefined && (
              <Button
                onClick={handleSaveSettings}
              >
                <CheckCircleIcon className="w-4 h-4" /> Save Settings
              </Button>
            )}
          </div>
        </SheetContent>
      </Sheet>

      <div className="flex w-full justify-between">
        <div className="flex gap-2">
          <SearchSelect
            options={providerDropdownItems}
            value={currentProviderId}
            onChange={handleProviderChange}
            placeholder="Select Provider"
            side="right"
            align="start"
            extraOptions={
              <CommandItem
                key="__add__"
                value="__add__"
                onSelect={() => setAddProviderOpen(true)}
              >
                <span className="flex items-center gap-2 text-green-500">
                  <PlusIcon className="w-4 h-4 stroke-green-500" />
                  Add a provider
                </span>
              </CommandItem>
            }
          />
          <SearchSelect
            disabled={!currentProviderId || models.length === 0}
            options={modelDropdownItems}
            value={currentModel || undefined}
            onChange={handleModelChange}
            placeholder="Select Model"
            side="right"
            align="start"
          />
        </div>
        <div className="flex gap-2">
          {currentProvider && (
            <Button variant="secondary" onClick={handleOpenSettings}>
              <CogIcon className="w-4 h-4" /> Settings
            </Button>
          )}
          {currentProvider && currentProvider.isEnvironmentDefined === false && (
            <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="secondary">
                <TrashIcon className="w-4 h-4" /> Delete Provider
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Provider</AlertDialogTitle>
                <AlertDialogDescription>
                  Are you sure you want to delete "{currentProvider.name}"? This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction asChild>
                  <Button
                    onClick={() => handleDeleteProvider(currentProvider.id)}
                    variant="destructive"
                  >
                    Delete
                  </Button>
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
          )}
        </div>
      </div>

      {!disableNewChat && (
        <div className="flex items-center">
          <Button onClick={handleClear} disabled={loadingProviders} variant="secondary">
            <ArrowPathIcon className="w-4 h-4" /> New Chat
          </Button>
        </div>
      )}
    </div>
  );
};