import type { FC} from 'react';
import { useCallback, useState } from 'react';
import { CheckCircleIcon, DocumentDuplicateIcon } from './heroicons';
import { copyToClipboard } from '../services/clipboard';

export const MessageCopyAction: FC<{ text: string }> = ({ text }) => {
    const [copied, setCopied] = useState(false);
    const handleCopy = useCallback(() => {
        void copyToClipboard(text).then(success => {
            if (success) {
                setCopied(true);
                setTimeout(() =>{  setCopied(false); }, 2000);
            }
        });
    }, [text]);
    return (
        <div className="flex items-center gap-1 opacity-0 group-hover/msg:opacity-100 transition-opacity mt-1">
            <button onClick={handleCopy} className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors" title="Copy">
                {copied
                    ? <CheckCircleIcon className="w-3.5 h-3.5 text-green-500" />
                    : <DocumentDuplicateIcon className="w-3.5 h-3.5" />
                }
            </button>
        </div>
    );
};
