// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import classNames from "classnames";
import { ChangeEvent, ChangeEventHandler, DetailedHTMLProps, FC, InputHTMLAttributes, KeyboardEventHandler, cloneElement, useCallback, useState } from "react";
import { twMerge } from "tailwind-merge";
import { Icons } from "./icons";


export const Text: FC<{ label: string }> = ({ label }) => {
    return <span className="text-xs text-gray-600 dark:text-gray-300">{label}</span>
}

export const Label: FC<{ label: string }> = ({ label }) => {
    return <strong><label className="text-xs text-gray-600 mt-2 dark:text-gray-300">{label}</label></strong>
}

type InputProps = {
    inputProps?: DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>;
    placeholder?: string;
    value: string;
    setValue?: (value: string) => void;
    type?: "text" | "password";
    onSubmit?: () => void;
}

export const Input: FC<InputProps> = ({ value, setValue, type, placeholder, onSubmit, inputProps = {} }) => {
    const handleChange: ChangeEventHandler<HTMLInputElement> = useCallback((e) => {
        setValue?.(e.target.value);
        inputProps.onChange?.(e);
    }, [inputProps, setValue]);

    const handleHandleKeyUp: KeyboardEventHandler<HTMLInputElement> = useCallback((e) => {
        if (e.key === "Enter") {
            onSubmit?.();
        }
        inputProps?.onKeyUp?.(e);
    }, [inputProps, onSubmit]);

    return <input type={type} placeholder={placeholder}
        {...inputProps} onChange={handleChange} value={value} onKeyUp={handleHandleKeyUp}
        className={twMerge(classNames("appearance-none border border-gray-200 rounded-md w-full p-1 text-gray-700 leading-tight focus:outline-none focus:shadow-outline text-sm h-[34px] px-2 dark:text-neutral-300/100 dark:bg-white/10 dark:border-white/20", inputProps.className))} />
}

type InputWithLabelProps = {
    label: string;
} & InputProps;

export const InputWithlabel: FC<InputWithLabelProps> = ({ value, setValue, label, type = "text", placeholder = `Enter ${label.toLowerCase()}`, inputProps }) => {
    const [hide, setHide] = useState(true);

    const handleShow = useCallback(() => {
        setHide(status => !status);
    }, []);

    const inputType = type === "password" ? hide ? "password" : "text" : type;

    return <div className="flex flex-col gap-1">
        <Label label={label} />
        <div className="relative">
            <Input type={inputType} value={value} setValue={setValue} inputProps={inputProps} placeholder={placeholder} />
            {type === "password" && cloneElement(hide ? Icons.Show : Icons.Hide, {
                className: "w-4 h-4 absolute right-2 top-1/2 -translate-y-1/2 cursor-pointer transition-all hover:scale-110 dark:stroke-neutral-300",
                onClick: handleShow,
            })}
        </div>
    </div>
}

type IToggleInputProps = {
    value: boolean;
    setValue: (value: boolean) => void;
}

export const ToggleInput: FC<IToggleInputProps> = ({ value, setValue }) => {
    const handleChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
        setValue(e.target.checked);
    }, [setValue]);

    return (
        <label className="inline-flex items-center cursor-pointer scale-75">
            <input type="checkbox" checked={value} className="sr-only peer" onChange={handleChange} />
            <div className="relative w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-[#ca6f1e]"></div>
        </label>
    );
}


type ICheckBoxInputProps = {
    value: boolean;
    setValue?: (value: boolean) => void;
}

export const CheckBoxInput: FC<ICheckBoxInputProps> = ({ value, setValue }) => {
    const handleChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
        setValue?.(e.target.checked);
    }, [setValue]);

    return (
        <input className="hover:cursor-pointer accent-[#ca6f1e] dark:accent-[#ca6f1e]" type="checkbox" checked={value} onChange={handleChange} />
    );
}
