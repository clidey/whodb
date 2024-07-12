import { DetailedHTMLProps, FC, InputHTMLAttributes, cloneElement } from "react";
import { Icons } from "./icons";
import { Input } from "./input";


type ISearchInputProps = {
    search: string;
    setSearch: (search: string) => void;
    placeholder?: string;
    inputProps?: DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>;
}

export const SearchInput: FC<ISearchInputProps> = ({ search, setSearch, placeholder, inputProps }) => {
    return (<div className="relative grow group/search-input">
        <Input value={search} setValue={setSearch} placeholder={placeholder} inputProps={{ autoFocus: true, ...inputProps }} />
        {cloneElement(Icons.Search, {
            className: "w-4 h-4 absolute right-2 top-1/2 -translate-y-1/2 stroke-gray-500 dark:stroke-neutral-500 cursor-pointer transition-all hover:scale-110 rounded-full group-hover/search-input:opacity-10",
        })}
    </div>)
}