/*
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

import {FC} from "react";
import {Container} from "./page";

import {Spinner} from "@clidey/ux";

type ILoadingProps = {
  className?: string;
  size?: "sm" | "md" | "lg";
  hideText?: boolean;
  loadingText?: string;
}

export const Loading: FC<ILoadingProps> = ({className, size = "md", hideText, loadingText}) => {
  let textSize = "text-base";
  if (size === "sm") {
      textSize = "text-xs";
  } else if (size === "md") {
      textSize = "text-sm";
  } else if (size === "lg") {
      textSize = "text-base";
  }

    return (
        <div className="flex justify-center items-center w-fit h-fit gap-sm" data-testid="loading-spinner">
            <Spinner className={className} size={size}/>
            {!hideText && <p className={textSize}>{loadingText}</p>}
        </div>
    );
};


export const LoadingPage: FC = () => {
  return <Container className="flex justify-center items-center h-full w-full">
    <Loading size="lg" />
  </Container>
}
