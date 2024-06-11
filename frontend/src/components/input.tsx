import classNames from "classnames";
import { ChangeEventHandler, DetailedHTMLProps, FC, InputHTMLAttributes, KeyboardEvent, cloneElement, useCallback, useState } from "react";
import { twMerge } from "tailwind-merge";
import { Icons } from "./icons";

export const Label: FC<{ label: string }> = ({ label }) => {
    return <strong><label className="text-xs text-gray-600 mt-2">{label}</label></strong>
}

type InputProps = {
    inputProps?: DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>;
    placeholder?: string;
    value: string;
    setValue?: (value: string) => void;
    type?: "text" | "password";
    multiline?: boolean;
    autoHeight?: boolean;
    onSubmit?: () => void;
}

export const Input: FC<InputProps> = ({ value, setValue, type, placeholder, multiline, inputProps = {}, autoHeight, onSubmit }) => {
    const handleChange: ChangeEventHandler<HTMLInputElement> = useCallback((e) => {
        setValue?.(e.target.value);
        inputProps.onChange?.(e);
    }, [inputProps, setValue]);

    const handleKeyDown = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
        inputProps.onKeyDown?.(e);
    }, [inputProps]);
    
    return <input type={type} placeholder={placeholder}
        value={value}  {...inputProps} onChange={handleChange} onKeyDown={handleKeyDown}
        className={twMerge(classNames("appearance-none border border-gray-200 rounded w-full p-1 text-gray-700 leading-tight focus:outline-none focus:shadow-outline text-sm h-[34px] px-2", inputProps.className))} />
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
                className: "w-4 h-4 absolute right-2 top-1/2 -translate-y-1/2 cursor-pointer transition-all hover:scale-110",
                onClick: handleShow,
            })}
        </div>
    </div>
}
