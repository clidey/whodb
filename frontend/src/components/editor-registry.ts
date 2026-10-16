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

// Kept free of CodeMirror imports so registering editor extensions at boot does not pull
// the editor into the main bundle.

type SQLAutocompleteFactory = (options: { apolloClient: any; defaultSchema?: string }) => any[];

/** Editor extensions registered by extensions and read by the code editor when it renders. */
export const editorExtensions: {
    createSQLAutocomplete?: SQLAutocompleteFactory;
    languages: Record<string, () => any>;
} = { languages: {} };

const extensionLoaders: Array<() => Promise<unknown>> = [];
let extensionsLoaded: Promise<void> | undefined;

/** Registers SQL autocomplete and additional editor languages. */
export const registerEditorExtensions = (fns: {
    createSQLAutocomplete?: SQLAutocompleteFactory;
    languages?: Record<string, () => any>;
}) => {
    if (fns.createSQLAutocomplete) {
        editorExtensions.createSQLAutocomplete = fns.createSQLAutocomplete;
    }
    if (fns.languages) {
        Object.assign(editorExtensions.languages, fns.languages);
    }
};

/**
 * Registers a loader that runs before the first code editor renders, so editor extensions
 * can live in a lazily loaded chunk. The loader is expected to call registerEditorExtensions.
 */
export const registerEditorExtensionsLoader = (loader: () => Promise<unknown>) => {
    extensionLoaders.push(loader);
};

/** Runs every registered editor extension loader once and resolves when all have finished. */
export const loadEditorExtensions = (): Promise<void> => {
    extensionsLoaded ??= Promise.all(extensionLoaders.map((load) => load())).then(() => undefined);
    return extensionsLoaded;
};
