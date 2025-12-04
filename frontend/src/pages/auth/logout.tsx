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

import { useMutation } from "@apollo/client";
import { FC, useEffect } from "react";
import { useDispatch } from "react-redux";
import { Container } from "../../components/page";
import { LogoutDocument, LogoutMutation, LogoutMutationVariables } from '@graphql';
import { AuthActions } from "../../store/auth";
import { Loading } from "../../components/loading";
import { toast } from "@clidey/ux";
import { useTranslation } from '@/hooks/use-translation';

export const LogoutPage: FC = () => {
  const { t } = useTranslation('pages/logout');
  const [logout, ] = useMutation<LogoutMutation, LogoutMutationVariables>(LogoutDocument);
  const dispatch = useDispatch();

  useEffect(() => {
    logout({
      onCompleted() {
        dispatch(AuthActions.logout());
        toast.success(t('success'));
      },
      onError() {
        toast.error(t('error'));
      }
    });
  }, [dispatch, logout, t]);

  return <Container>
      <div className="flex flex-col justify-center items-center gap-lg w-full">
          <div>
              <Loading hideText={true} />
          </div>
          <div className="text-neutral-800 dark:text-neutral-300">
              {t('loggingOut')}
          </div>
      </div>
  </Container>
}