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

const DEFAULT_TITLE = "Clidey WhoDB";
const DEFAULT_ICON = "/images/logo.svg";

export const updateDocumentMeta = (extensions: Record<string, any>) => {
    const title = extensions.MetaTitle || DEFAULT_TITLE;
    const icon = extensions.MetaIcon || DEFAULT_ICON;

    document.title = title;

    const faviconLink = document.querySelector("link[rel='icon']") as HTMLLinkElement;
    if (faviconLink) {
        faviconLink.href = icon;
    }

    const appleTouchIcon = document.querySelector("link[rel='apple-touch-icon']") as HTMLLinkElement;
    if (appleTouchIcon) {
        appleTouchIcon.href = icon;
    }
};
