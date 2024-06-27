import { isNaN, startCase, toLower } from "lodash";

export function isNumeric(str: string) {
    return !isNaN(Number(str));
}

export function createStub(name: string) {
    return name.split(" ").map(word => word.toLowerCase()).join("-");
}

export function toTitleCase(str: string) {
    return startCase(toLower(str));
}