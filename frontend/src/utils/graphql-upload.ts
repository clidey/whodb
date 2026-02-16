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

import { addAuthHeader } from "@/utils/auth-headers";

export type GraphQLUploadRequest = {
  query: string;
  variables: Record<string, any>;
  file: File;
  filePath: string;
  operationName?: string;
};

/**
 * Executes a multipart GraphQL request for a single file upload.
 */
export async function executeMultipartGraphQL<T>({
  query,
  variables,
  file,
  filePath,
  operationName,
}: GraphQLUploadRequest): Promise<T> {
  const formData = new FormData();
  formData.append(
    "operations",
    JSON.stringify({
      query,
      variables,
      operationName,
    }),
  );
  formData.append("map", JSON.stringify({ "0": [filePath] }));
  formData.append("0", file);

  const response = await fetch("/api/query", {
    method: "POST",
    credentials: "include",
    headers: addAuthHeader({}),
    body: formData,
  });

  const result = await response.json();
  if (!response.ok || result.errors?.length) {
    const message = result.errors?.[0]?.message ?? "";
    throw new Error(message);
  }

  return result.data as T;
}
