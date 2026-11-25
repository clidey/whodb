/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { SheetFooter, SheetTitle, cn } from '@clidey/ux';
import type { FC, ReactNode } from 'react';

type SheetHeadingProps = {
    /** Small uppercase line above the title naming what the drawer acts on, such as the table. */
    context?: ReactNode;
    title: ReactNode;
    className?: string;
};

/** Standard side-drawer heading: an optional context line above the drawer title. */
export const SheetHeading: FC<SheetHeadingProps> = ({ context, title, className }) => (
    <div className={cn('ce-sheet-heading', className)}>
        {context != null && context !== '' && <span>{context}</span>}
        <SheetTitle>{title}</SheetTitle>
    </div>
);

type SheetActionsProps = {
    /** Short status shown on the left of the footer, such as a count or file name. */
    hint?: ReactNode;
    children: ReactNode;
    className?: string;
};

/** Standard side-drawer footer: an optional hint on the left, then Cancel and the main action on the right. */
export const SheetActions: FC<SheetActionsProps> = ({ hint, children, className }) => (
    <SheetFooter className={cn('ce-sheet-footer', className)}>
        {hint != null && hint !== '' && <span className="ce-sheet-footer-hint">{hint}</span>}
        {children}
    </SheetFooter>
);
