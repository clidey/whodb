import { FC } from 'react';
import { Alert, AlertDescription, AlertTitle, Button, Spinner } from '@clidey/ux';
import { useTranslation } from '@/hooks/use-translation';
import { useErrorExplanation } from '../hooks/use-error-explanation';
import { SparklesIcon } from './heroicons';

const rainbowStyle = {
    background: 'linear-gradient(90deg, #f43f5e, #f59e0b, #22c55e, #3b82f6, #a855f7, #f43f5e)',
    backgroundSize: '200% auto',
    animation: 'rainbow-shift 3s linear infinite',
    WebkitBackgroundClip: 'text',
    WebkitTextFillColor: 'transparent',
    backgroundClip: 'text',
} as const;

const rainbowKeyframes = `@keyframes rainbow-shift { 0% { background-position: 0% center; } 100% { background-position: 200% center; } }`;

interface ErrorExplanationProps {
    error: string;
    query: string;
    dbType: string;
    schema: string;
}

export const ErrorExplanation: FC<ErrorExplanationProps> = ({ error, query, dbType, schema }) => {
    const { t } = useTranslation('components/error-explanation');
    const { enabled, status, result, progress, errorMessage, retry } = useErrorExplanation({
        error,
        query,
        dbType,
        schema,
    });

    if (!enabled || status === 'idle') return null;

    if (status === 'loading-model') {
        return (
            <Alert className="mt-2" data-testid="error-explanation-loading">
                <style>{rainbowKeyframes}</style>
                <SparklesIcon className="w-4 h-4" />
                <AlertTitle style={rainbowStyle}>{t('title')}</AlertTitle>
                <AlertDescription>
                    <div className="flex items-center gap-2">
                        <Spinner />
                        <span>
                            {progress != null && progress > 0
                                ? t('downloadingProgress', { progress: Math.round(progress) })
                                : t('downloading')}
                        </span>
                    </div>
                </AlertDescription>
            </Alert>
        );
    }

    if (status === 'generating') {
        return (
            <Alert className="mt-2" data-testid="error-explanation-generating">
                <style>{rainbowKeyframes}</style>
                <SparklesIcon className="w-4 h-4" />
                <AlertTitle style={rainbowStyle}>{t('title')}</AlertTitle>
                <AlertDescription>
                    <div className="flex items-center gap-2">
                        <Spinner />
                        <span>{t('generating')}</span>
                    </div>
                </AlertDescription>
            </Alert>
        );
    }

    if (status === 'error') {
        return (
            <Alert variant="destructive" className="mt-2" data-testid="error-explanation-error">
                <SparklesIcon className="w-4 h-4" />
                <AlertTitle>{t('title')}</AlertTitle>
                <AlertDescription>
                    <div className="flex items-center justify-between">
                        <span>{errorMessage ?? t('unknownError')}</span>
                        <Button variant="outline" size="sm" onClick={retry}>{t('retry')}</Button>
                    </div>
                </AlertDescription>
            </Alert>
        );
    }

    if (status === 'done' && result) {
        return (
            <Alert className="mt-2" data-testid="error-explanation-result">
                <style>{rainbowKeyframes}</style>
                <SparklesIcon className="w-4 h-4" />
                <AlertTitle style={rainbowStyle}>{t('title')}</AlertTitle>
                <AlertDescription>
                    <div className="flex flex-col gap-2" style={rainbowStyle}>
                        <p>{result.explanation}</p>
                        {result.fix && (
                            <div className="rounded-md bg-muted p-2" style={{ WebkitTextFillColor: 'initial', backgroundClip: 'initial', WebkitBackgroundClip: 'initial', background: 'var(--muted)' }}>
                                <p className="text-xs font-medium mb-1">{t('suggestedFix')}</p>
                                <pre className="text-xs whitespace-pre-wrap"><code>{result.fix}</code></pre>
                            </div>
                        )}
                    </div>
                </AlertDescription>
            </Alert>
        );
    }

    return null;
};
