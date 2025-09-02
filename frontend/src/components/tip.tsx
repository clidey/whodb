import { Tooltip, TooltipContent, TooltipTrigger } from "@clidey/ux";
import { FC, ReactNode } from "react";

export const Tip: FC<{
    children: [ReactNode, ReactNode]
}> = ({ children }) => {
    return (
        <Tooltip>
            <TooltipTrigger className="w-fit">
                {children[0]}
            </TooltipTrigger>
            <TooltipContent>
                {children[1]}
            </TooltipContent>
        </Tooltip>
    )
}