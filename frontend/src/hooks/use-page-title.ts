import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

const DEFAULT_APP_NAME = 'WhoDB';

let appName = DEFAULT_APP_NAME;

export const setAppName = (name: string): void => {
    appName = name || DEFAULT_APP_NAME;
};

export const usePageTitle = (pageTitle?: string): void => {
    useEffect(() => {
        document.title = pageTitle ? `${pageTitle} - ${appName}` : appName;
        return () => { document.title = appName; };
    }, [pageTitle]);
};

const routeTitles: [RegExp, string][] = [
    [/\/login/, 'Login'],
    [/\/chat/, 'Chat'],
    [/\/graph/, 'Graph'],
    [/\/scratchpad/, 'Scratchpad'],
    [/\/settings/, 'Settings'],
    [/\/storage-unit\/explore/, 'Explore'],
    [/\/storage-unit/, 'Storage Unit'],
];

export const registerRouteTitles = (titles: [RegExp, string][]): void => {
    routeTitles.unshift(...titles);
};

export const PageTitleUpdater = (): null => {
    const { pathname } = useLocation();
    const title = routeTitles.find(([pattern]) => pattern.test(pathname))?.[1];
    usePageTitle(title);
    return null;
};
