import type { CSSProperties, FC, MouseEventHandler } from 'react';
import { cn } from '@clidey/ux';

import { icons } from '@clidey/ux/icons';

type WhoDBChatIconProps = {
    name: 'chat' | 'plus' | 'plus-circle' | 'search' | 'send' | 'table' | 'relation' | 'code' | 'source' | 'sidebar' | 'chevron-up' | 'chevron-down' | 'chevron-left' | 'chevron-right' | 'mail' | 'settings' | 'signout' | 'secret' | 'sliders' | 'hash' | 'banknotes' | 'play' | 'upload' | 'download' | 'help' | 'home' | 'moon' | 'sun' | 'calendar' | 'clock' | 'check-circle' | 'copy' | 'file-text' | 'globe' | 'grid' | 'list';
    className?: string;
    onClick?: MouseEventHandler<HTMLSpanElement>;
};

/** Renders the shared WhoDB chat icons in the current text colour. */
export const WhoDBChatIcon: FC<WhoDBChatIconProps> = ({ name, className, onClick }) => {
    const url = icons[name];
    const style: CSSProperties = {
        backgroundColor: 'currentColor',
        maskImage: `url("${url}")`,
        WebkitMaskImage: `url("${url}")`,
        maskPosition: 'center',
        WebkitMaskPosition: 'center',
        maskRepeat: 'no-repeat',
        WebkitMaskRepeat: 'no-repeat',
        maskSize: 'contain',
        WebkitMaskSize: 'contain',
    };

    return <span aria-hidden="true" data-whodb-icon={name} className={cn('inline-block size-4 shrink-0', className)} style={style} onClick={onClick} />;
};
