import {
    AutoTokenizer,
    AutoModelForCausalLM,
    env as transformersEnv,
    type PreTrainedTokenizer,
    type PreTrainedModel,
} from '@huggingface/transformers';

// TODO: Switch to fine-tuned model once proper q4f16 ONNX conversion is done
// Fine-tuned repo: ahhhhhh3/whodb-error-explainer-qwen25-coder-ONNX
// Blocked on: Transformers.js q4f16 quantizer not available as standalone tooling
const MODEL_ID = 'onnx-community/Qwen2.5-Coder-0.5B-Instruct';

const SYSTEM_PROMPT =
    'You are a database error explainer for WhoDB. ' +
    'Given an error message, the user\'s query, and schema context, ' +
    'provide a clear explanation and suggested fix. ' +
    'Respond directly without any internal reasoning. /no_think';

type LoadMessage = { type: 'load' };
type ExplainMessage = {
    type: 'explain';
    id: string;
    error: string;
    query: string;
    dbType: string;
    schema: string;
};
type IncomingMessage = LoadMessage | ExplainMessage;

let tokenizer: PreTrainedTokenizer | null = null;
let model: PreTrainedModel | null = null;
let loading: Promise<void> | null = null;

let lastProgressTime = 0;
const progressCallback = (progress: { status: string; progress?: number; file?: string }) => {
    const now = Date.now();
    if (now - lastProgressTime < 500 && progress.status === 'progress') return;
    lastProgressTime = now;
    self.postMessage({ type: 'progress', ...progress });
};

async function loadModel(): Promise<void> {
    if (model && tokenizer) return;
    if (loading) return loading;

    loading = (async () => {
        self.postMessage({ type: 'status', status: 'loading-model' });

        console.log(`[error-explainer] loading tokenizer from ${MODEL_ID}…`);
        tokenizer = await AutoTokenizer.from_pretrained(MODEL_ID, {
            progress_callback: progressCallback,
        });

        console.log(`[error-explainer] loading model (q4f16 + webgpu)…`);
        model = await AutoModelForCausalLM.from_pretrained(MODEL_ID, {
            dtype: 'q4f16',
            device: 'webgpu',
            progress_callback: progressCallback,
        });
        console.log(`[error-explainer] model loaded`);

        console.log('[error-explainer] warming up…');
        const warmupInputs = tokenizer('a');
        await model.generate({ ...warmupInputs, max_new_tokens: 1 });

        console.log(`[error-explainer] ready`);
        self.postMessage({ type: 'status', status: 'ready' });
    })();

    loading.catch(() => { loading = null; });
    return loading;
}

async function explain(msg: ExplainMessage): Promise<void> {
    await loadModel();

    self.postMessage({ type: 'status', status: 'generating', id: msg.id });

    const messages = [
        { role: 'system', content: SYSTEM_PROMPT },
        {
            role: 'user',
            content:
                `Database: ${msg.dbType}\n` +
                `Schema: ${msg.schema}\n` +
                `Query: ${msg.query}\n` +
                `Error: ${msg.error}`,
        },
    ];

    const inputs = tokenizer!.apply_chat_template(messages, {
        add_generation_prompt: true,
        return_dict: true,
    });

    const inputLength = (inputs as any).input_ids.dims[1];

    const outputIds = await model!.generate({
        ...(inputs as any),
        max_new_tokens: 256,
        temperature: 0.3,
        top_p: 0.9,
        do_sample: true,
        repetition_penalty: 1.05,
    });

    const generatedIds = (outputIds as any).slice(null, [inputLength, null]);
    const decoded = tokenizer!.batch_decode(generatedIds, { skip_special_tokens: true });
    const assistantText = (decoded[0] ?? '').trim();

    console.log('[error-explainer] generated:', assistantText);
    self.postMessage({ type: 'result', id: msg.id, text: assistantText });
}

self.addEventListener('message', (e: MessageEvent<IncomingMessage>) => {
    const msg = e.data;
    if (msg.type === 'load') {
        loadModel().catch((err) => {
            console.error('[error-explainer] load failed:', err);
            self.postMessage({ type: 'error', message: String(err) });
        });
    } else if (msg.type === 'explain') {
        explain(msg).catch((err) => {
            console.error('[error-explainer] explain failed:', err);
            self.postMessage({ type: 'error', id: msg.id, message: String(err) });
        });
    }
});
