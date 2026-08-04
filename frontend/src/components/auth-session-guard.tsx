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

import { useLazyQuery } from '@apollo/client/react';
import { Spinner } from '@clidey/ux';
import type { FC, ReactNode } from 'react';
import { useEffect, useState } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { SourceSessionDocument } from '../generated/graphql';
import { AuthActions } from '../store/auth';
import { useAppDispatch } from '../store/hooks';

interface AuthSessionGuardProps {
  children: ReactNode;
}

/** Restores CE UI auth state from the backend-owned browser session. */
export const AuthSessionGuard: FC<AuthSessionGuardProps> = ({ children }) => {
  const dispatch = useAppDispatch();
  const location = useLocation();
  const [checking, setChecking] = useState(true);
  const [hasSession, setHasSession] = useState(false);
  const [loadSourceSession] = useLazyQuery(SourceSessionDocument, { fetchPolicy: 'no-cache' });

  useEffect(() => {
    let mounted = true;
    setChecking(true);
    void loadSourceSession().then(({ data }) => {
      if (!mounted) {
        return;
      }
      const session = data?.SourceSession;
      if (!session) {
        dispatch(AuthActions.logout());
        setHasSession(false);
        setChecking(false);
        return;
      }

      dispatch(AuthActions.login({
        Id: session.id ?? `session:${session.sourceType}`,
        SourceType: session.sourceType,
        Values: session.database ? [{ Key: 'Database', Value: session.database }] : [],
      }));
      setHasSession(true);
      setChecking(false);
    }).catch(() => {
      if (!mounted) {
        return;
      }
      dispatch(AuthActions.logout());
      setHasSession(false);
      setChecking(false);
    });

    return () => {
      mounted = false;
    };
  }, [dispatch, loadSourceSession, location.pathname, location.search]);

  if (checking) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Spinner className="w-8 h-8" />
      </div>
    );
  }

  if (!hasSession) {
    return <Navigate to="/login" replace />;
  }

  return children;
};
