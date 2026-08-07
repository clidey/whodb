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
import { useEffect, useRef, useState } from 'react';
import { Navigate } from 'react-router-dom';
import { SourceSessionDocument } from '../generated/graphql';
import { AuthActions } from '../store/auth';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import { isDesktopApp } from '../utils/external-links';

interface AuthSessionGuardProps {
  children: ReactNode;
}

/**
 * Restores CE UI auth state from the backend-owned browser session.
 *
 * Runs once on mount, not on every navigation: this component wraps
 * `PrivateRoute`'s `<Outlet/>` and stays mounted across in-app navigation, so
 * re-running the session check on every route change would race with (and
 * clobber) in-flight profile/database switches, which dispatch a richer
 * profile object than the minimal one built from `SourceSession`. Session
 * expiry mid-session is already handled by the GraphQL error link's 401
 * auto-login.
 */
export const AuthSessionGuard: FC<AuthSessionGuardProps> = ({ children }) => {
  const dispatch = useAppDispatch();
  const loggedIn = useAppSelector(state => state.auth.status === 'logged-in');
  const desktopApp = isDesktopApp();
  const [checking, setChecking] = useState(true);
  const [hasSession, setHasSession] = useState(false);
  const [loadSourceSession] = useLazyQuery(SourceSessionDocument, { fetchPolicy: 'no-cache' });
  const hasChecked = useRef(false);

  useEffect(() => {
    if (desktopApp) {
      setChecking(false);
      setHasSession(loggedIn);
      return;
    }
    if (hasChecked.current) {
      return;
    }
    hasChecked.current = true;

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
  }, [desktopApp, dispatch, loadSourceSession, loggedIn]);

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
