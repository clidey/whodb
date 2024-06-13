import classNames from "classnames";
import { FC, ReactElement, useCallback, useState } from "react";
import { Icons } from "./icons";
import { Label } from "./input";
import { Loading } from "./loading";

export type IDropdownItem = {
    id: string;
    label: string;
    icon?: ReactElement;
};

export type IDropdownProps = {
    className?: string;
    items: IDropdownItem[];
    loading?: boolean;
    value?: IDropdownItem;
    onChange?: (item: IDropdownItem) => void;
}

export const Dropdown: FC<IDropdownProps> = (props) => {
    const [hover, setHover] = useState(false);

    const handleClick = useCallback((item: IDropdownItem) => {
        setHover(false);
        props.onChange?.(item);
    }, [props]);

    const handleMouseEnter = useCallback(() => {
        setHover(true);
    }, []);

    const handleMouseLeave = useCallback(() => {
        setHover(false);
    }, []);

    return (
        <button className={classNames("relative", props.className)} onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave}>
            {props.loading ? <Loading hideText={true} /> : 
            <>  <div className="flex items-center border border-gray-200 rounded w-full p-1 text-gray-700 text-sm h-[34px] px-2 gap-1">
                    {props.value?.icon} {props.value?.label}
                </div>
                <div className={classNames("absolute w-full z-10 divide-y rounded-lg shadow bg-white py-1 border border-gray-200 overflow-y-auto max-h-40", {
                    "hidden": !hover,
                    "block animate-fade": hover,
                })}>
                    <ul className="py-1 text-sm text-gray-700 nowheel flex flex-col">
                        {
                            props.items.map((item) => (
                                <li key={item.id} className={classNames("group/item flex items-center gap-1 transition-all cursor-pointer relative hover:bg-black/10 py-1 mx-2 rounded-lg pl-1", {
                                    "hover:gap-2": item.icon != null,
                                })} onClick={() => handleClick(item)}>
                                    <div>{props.value?.id === item.id ? Icons.CheckCircle : item.icon}</div>
                                    <div>{item.label}</div>
                                </li>
                            ))
                        }
                    </ul>
                </div>
            </>}
        </button>
    )
}

export const DropdownWithLabel: FC<IDropdownProps & { label: string }> = ({ label, ...props }) => {
    return <div className="flex flex-col gap-1">
        <Label label={label} />
        <Dropdown {...props} />
    </div>
}