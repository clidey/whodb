/**
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

import { DetailedHTMLProps, FC, InputHTMLAttributes, cloneElement } from "react";
import { Icons } from "./icons";
import { Input } from "./input";


type ISearchInputProps = {
    search: string;
    setSearch: (search: string) => void;
    placeholder?: string;
    inputProps?: DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>;
    testId?: string;
}

export const SearchInput: FC<ISearchInputProps> = ({ search, setSearch, placeholder, inputProps, testId }) => {
    return (<div className="relative grow group/search-input" data-testid={testId}>
        <Input value={search} setValue={setSearch} placeholder={placeholder} inputProps={{ autoFocus: true, ...inputProps, }} />
        {cloneElement(Icons.Search, {
            className: "w-4 h-4 absolute right-2 top-1/2 -translate-y-1/2 stroke-gray-500 dark:stroke-neutral-500 cursor-pointer transition-all hover:scale-110 rounded-full group-hover/search-input:opacity-10",
        })}
    </div>)
}