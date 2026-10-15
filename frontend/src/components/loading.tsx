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

import type {FC, SVGProps} from "react";

import {cn} from "@clidey/ux";
import {useTranslation} from "@/hooks/use-translation";

type ILoadingProps = {
  className?: string;
  size?: "sm" | "md" | "lg";
  hideText?: boolean;
  loadingText?: string;
}

/** Animated WhoDB mark sized for buttons, tables, panels, and inline status. */
export const Spinner: FC<SVGProps<SVGSVGElement>> = ({className, ...props}) => {
  const {t} = useTranslation('components/loading');
  const hidden = props['aria-hidden'] === true || props['aria-hidden'] === 'true';

  return <svg
    viewBox="0 0 120 96"
    fill="none"
    role={hidden ? undefined : 'status'}
    aria-label={hidden ? undefined : t('loading')}
    className={cn('whodb-loader-mark size-5 shrink-0', className)}
    {...props}
  >
    <path className="whodb-loader-mark__outline" d="M18 20 C24 34 42 63 60 63 C78 63 96 34 102 20" />
    <path className="whodb-loader-mark__stem-outline" d="M60 29 V80" />
    <path className="whodb-loader-mark__signal" pathLength="100" d="M18 20 C24 34 42 63 60 63" />
    <path className="whodb-loader-mark__signal" pathLength="100" d="M102 20 C96 34 78 63 60 63" />
    <path className="whodb-loader-mark__meet" d="M48 59.1 C52 61.6 56 63 60 63 C64 63 68 61.6 72 59.1" />
    <path className="whodb-loader-mark__stem" pathLength="100" d="M60 29 V80" />
    <circle className="whodb-loader-mark__ring" cx="18" cy="20" r="4" />
    <circle className="whodb-loader-mark__ring" cx="102" cy="20" r="4" />
    <circle className="whodb-loader-mark__node" cx="35" cy="48" r="2.5" />
    <circle className="whodb-loader-mark__node" cx="85" cy="48" r="2.5" />
    <circle className="whodb-loader-mark__beacon" cx="60" cy="18" r="6" />
  </svg>;
};

/** Reusable WhoDB loading mark for inline progress and page transitions. */
export const Loading: FC<ILoadingProps> = ({className, size = "md", hideText = true, loadingText}) => {
  const { t } = useTranslation('components/loading');
  let textSize = "text-base";
  if (size === "sm") {
      textSize = "text-xs";
  } else if (size === "md") {
      textSize = "text-sm";
  } else if (size === "lg") {
      textSize = "text-base";
  }

    return (
        <div
            className="flex justify-center items-center w-fit h-fit gap-sm"
            data-testid="loading-spinner"
            role="status"
            aria-busy="true"
            aria-label={loadingText ?? t('loading')}
        >
            <Spinner
                aria-hidden="true"
                className={cn(size === 'sm' ? 'size-5' : size === 'lg' ? 'size-8' : 'size-6', className)}
            />
            {!hideText && <p className={textSize}>{loadingText ?? t('loading')}</p>}
        </div>
    );
};


/** Full-page loading state with the animated WhoDB mark and status text. */
export const LoadingPage: FC = () => {
  const {t} = useTranslation('components/loading');
  return <div className="flex flex-col justify-center items-center gap-3 h-full w-full min-h-48">
    <Loading size="lg" className="size-32" loadingText={t('gettingThingsReady')} />
    <p className="text-xs text-muted-foreground" aria-hidden="true">{t('gettingThingsReady')}</p>
  </div>
}
