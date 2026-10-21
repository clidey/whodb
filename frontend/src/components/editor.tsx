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

import type { FC } from "react";
import { lazy, Suspense } from "react";
import type { ICodeEditorProps } from "./code-editor";
import { loadEditorExtensions } from "./editor-registry";

const LazyCodeEditor = lazy(async () => {
    const [module] = await Promise.all([import("./code-editor"), loadEditorExtensions()]);
    return { default: module.CodeEditor };
});

/**
 * Code editor that loads CodeMirror and any registered editor extensions on first render,
 * keeping them out of the main bundle.
 */
export const CodeEditor: FC<ICodeEditorProps> = (props) => (
    <Suspense fallback={<div className="h-full w-full" />}>
        <LazyCodeEditor {...props} />
    </Suspense>
);
