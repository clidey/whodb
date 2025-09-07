import { cn, Tooltip, TooltipContent, TooltipTrigger } from "@clidey/ux";
import { FC, ReactNode } from "react";

export const Tip: FC<{
    className?: string;
    children: [ReactNode, ReactNode]
}> = ({ children, className }) => {
    return (
        <Tooltip>
            <TooltipTrigger className={cn("w-full", className)}>
                {children[0]}
            </TooltipTrigger>
            <TooltipContent>
                {children[1]}
            </TooltipContent>
        </Tooltip>
    )
}