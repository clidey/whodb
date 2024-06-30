import { isNaN, startCase, toLower } from "lodash";
import { DatabaseType } from "../generated/graphql";

export function isNumeric(str: string) {
    return !isNaN(Number(str));
}

export function createStub(name: string) {
    return name.split(" ").map(word => word.toLowerCase()).join("-");
}

export function toTitleCase(str: string) {
    return startCase(toLower(str));
}

export function isMarkdown(text: string): boolean {
    const markdownPatterns = [
        /^#{1,6}\s+/,
        /^\s*[-*+]\s+/,
        /^\d+\.\s+/,
        /\*\*[^*]+\*\*/,
        /_[^_]+_/,
        /!\[.*?\]\(.*?\)/,
        /\[.*?\]\(.*?\)/,
        /^>\s+/,
        /`{1,3}[^`]*`{1,3}/,
        /-{3,}/,
    ];
    
    return markdownPatterns.some(pattern => pattern.test(text));
}

export function isValidJSON(str: string): boolean {
    // this allows it to start showing intellisense when it starts with {
    // even if it is not valid - better UX?
    return str.startsWith("{");
}

export function isNoSQL(databaseType: string) {
    switch (databaseType) {
        case DatabaseType.MongoDb:
            return true;
    }
    return false;
}