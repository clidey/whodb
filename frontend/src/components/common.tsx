// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import { FC, ReactElement, ReactNode } from "react";
import { createPortal } from "react-dom";

type IEmptyMessageProps = {
    icon: ReactElement;
    label: string;
}

export const EmptyMessage: FC<IEmptyMessageProps> = ({ icon, label }) => {
    return (
        <div className="flex gap-2 items-center justify-center absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 dark:text-neutral-300">
            {icon}
            {label}
        </div>
    )   
}

interface PortalProps {
  children: ReactNode;
}

export const Portal: FC<PortalProps> = ({ children }) => {
  return createPortal(children, document.querySelector("#whodb-app-container")!);
};
