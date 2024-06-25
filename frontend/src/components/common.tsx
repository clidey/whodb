import { FC, ReactElement } from "react";

type IEmptyMessageProps = {
    icon: ReactElement;
    label: string;
}

export const EmptyMessage: FC<IEmptyMessageProps> = ({ icon, label }) => {
    return (
        <div className="flex gap-2 items-center justify-center absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2">
            {icon}
            {label}
        </div>
    )   
}