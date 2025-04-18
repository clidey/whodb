/**
 * Copyright 2025 Clidey, Inc.
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

import classNames from "classnames";
import { FC } from "react";
import { twMerge } from "tailwind-merge";
import { ClassNames } from "./classes";
import { Container } from "./page";

type ILoadingProps = {
  className?: string;
  hideText?: boolean;
  loadingText?: string;
  size?: "lg" | "md" | "sm";
}

export const Loading: FC<ILoadingProps> = ({ className, hideText, loadingText, size = "md" }) => {
  if (size === "sm") {
    return <div className="flex justify-center items-center w-fit h-fit gap-1">
      <div className="h-[16px] w-[16px] relative">
        <div className="scale-[0.25] absolute top-0 left-0 -translate-y-[20px] -translate-x-[20px]">
          <Loading className={className} hideText={hideText} loadingText={loadingText} size="lg" />
        </div>
      </div>
      {
        !hideText &&
        <div className={classNames(ClassNames.Text, "text-sm")}>{loadingText}</div>
      }
    </div>
  }
  if (size === "md") {
    return <div className="flex justify-center items-center w-fit h-fit gap-1">
      <div className="h-[32px] w-[32px] relative">
        <div className="scale-[0.5] absolute top-0 left-0 -translate-y-[12px] -translate-x-[12px]">
          <Loading className={className} hideText={hideText} loadingText={loadingText} size="lg" />
        </div>
      </div>
      {
        !hideText &&
        <div className={classNames(ClassNames.Text, "text-sm")}>{loadingText}</div>
      }
    </div>
  }
  return <div className={twMerge("loader w-14 aspect-square animate-boxy rounded-full", className)}></div>;
}


export const LoadingPage: FC = () => {
  return <Container className="flex justify-center items-center h-full w-full">
    <Loading size="lg" />
  </Container>
}
