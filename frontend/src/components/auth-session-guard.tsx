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
 * Checks the session at most once, on mount, deliberately excluding
 * `loggedIn` from the effect's dependency array: this component wraps
 * `PrivateRoute`'s `<Outlet/>` and stays mounted across in-app navigation
 * *and* across login/logout, so a `loggedIn`-reactive effect would re-fire
 * on every `AuthActions.login()`/`logout()` dispatch — racing with (and
 * clobbering) in-flight profile/database switches, and re-showing the
 * loading spinner mid-navigation on `/logout` right as it navigates to
 * `/login` (observed as Playwright `net::ERR_ABORTED` on that navigation).
 * Once the mount check completes, redirect/render decisions read `loggedIn`
 * directly so they still reflect logins/logouts dispatched elsewhere — this
 * component just never re-fetches the session to do so. Session expiry
 * mid-session is already handled separately by the GraphQL error link's 401
 * auto-login.
 *
 * The mount check itself is skipped when Redux already reports a logged-in
 * profile — this component first mounts right after the login page's own
 * `AuthActions.login()` dispatch (with the full profile — Hostname,
 * Password, etc.), and `SourceSession` only carries Id/SourceType/Database.
 * Dispatching that minimal payload here would clobber the credentials the
 * login flow just stored, breaking any later profile/database switch that
 * needs them.
 */
export const AuthSessionGuard: FC<AuthSessionGuardProps> = ({ children }) => {
  const dispatch = useAppDispatch();
  const loggedIn = useAppSelector(state => state.auth.status === 'logged-in');
  const desktopApp = isDesktopApp();
  const [checking, setChecking] = useState(true);
  const [loadSourceSession] = useLazyQuery(SourceSessionDocument, { fetchPolicy: 'no-cache' });

  useEffect(() => {
    if (desktopApp || loggedIn) {
      setChecking(false);
      return;
    }

    let mounted = true;
    void loadSourceSession().then(({ data }) => {
      if (!mounted) {
        return;
      }
      const session = data?.SourceSession;
      if (!session) {
        dispatch(AuthActions.logout());
        setChecking(false);
        return;
      }

      dispatch(AuthActions.login({
        Id: session.id ?? `session:${session.sourceType}`,
        SourceType: session.sourceType,
        Values: session.database ? [{ Key: 'Database', Value: session.database }] : [],
      }));
      setChecking(false);
    }).catch(() => {
      if (!mounted) {
        return;
      }
      dispatch(AuthActions.logout());
      setChecking(false);
    });

    return () => {
      mounted = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally excludes `loggedIn`; see comment above
  }, [desktopApp, dispatch, loadSourceSession]);

  if (checking) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Spinner className="w-8 h-8" />
      </div>
    );
  }

  if (!loggedIn) {
    return <Navigate to="/login" replace />;
  }

  return children;
};
