import { SearchSelect as SearchSelectComponent  } from "@clidey/ux";
import { ChevronUpDownIcon } from "./heroicons";

export const SearchSelect = (props: React.ComponentProps<typeof SearchSelectComponent>) => {
    return <SearchSelectComponent {...props} rightIcon={<ChevronUpDownIcon className="w-4 h-4" />} />
}