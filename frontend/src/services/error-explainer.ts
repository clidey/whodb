export type ErrorExplainerListener = (event:
    | { type: 'status'; status: string }
    | { type: 'progress'; status: string; progress?: number; file?: string }
    | { type: 'result'; id: string; text: string }
    | { type: 'error'; id?: string; message: string }
) => void;

let worker: Worker | null = null;
let modelReady = false;
let listeners = new Set<ErrorExplainerListener>();

function getWorker(): Worker {
    if (!worker) {
        worker = new Worker(
            new URL('../workers/error-explainer.worker.ts', import.meta.url),
            { type: 'module' },
        );
        worker.addEventListener('message', (e) => {
            if (e.data.type === 'status' && e.data.status === 'ready') {
                modelReady = true;
            }
            for (const listener of listeners) {
                listener(e.data);
            }
        });
    }
    return worker;
}

/** Start downloading the model. Idempotent — safe to call multiple times. */
export function preloadModel(): void {
    getWorker().postMessage({ type: 'load' });
}

/** Request an error explanation. Returns a unique ID to match with the result. */
export function requestExplanation(params: {
    id: string;
    error: string;
    query: string;
    dbType: string;
    schema: string;
}): void {
    getWorker().postMessage({ type: 'explain', ...params });
}

export function addListener(listener: ErrorExplainerListener): () => void {
    listeners.add(listener);
    return () => { listeners.delete(listener); };
}

/** Check if the model has finished loading. */
export function isModelReady(): boolean {
    return modelReady;
}

/** Terminate the worker and release memory. */
export function dispose(): void {
    if (worker) {
        worker.terminate();
        worker = null;
    }
    modelReady = false;
    listeners.clear();
}
