// Deliberately wrong trigger usage. Used only to prove check-trigger-children.mjs fails:
//   node scripts/check-trigger-children.mjs scripts/__fixtures__   (expected exit code 1)
// Not part of any build or lint target.
import { Button, CopyButton, DropdownMenuTrigger, PopoverTrigger, TooltipTrigger } from "@clidey/ux";

export function BadTriggers({ show }: { show: boolean }) {
    return (
        <>
            <TooltipTrigger asChild>Plain text</TooltipTrigger>
            <TooltipTrigger asChild>
                <>
                    <Button>A</Button>
                </>
            </TooltipTrigger>
            <DropdownMenuTrigger asChild>
                <Button>A</Button>
                <Button>B</Button>
            </DropdownMenuTrigger>
            <PopoverTrigger asChild>{show ? <Button>A</Button> : null}</PopoverTrigger>
            <PopoverTrigger asChild>{[<Button key="a">A</Button>]}</PopoverTrigger>
            <TooltipTrigger asChild>
                <CopyButton text="x" />
            </TooltipTrigger>
            <TooltipTrigger>
                <CopyButton text="x" />
            </TooltipTrigger>
        </>
    );
}
