import React from 'react';
import ReactDOM from 'react-dom/client';
import { ApolloProvider } from '@apollo/client';
import { graphqlClient } from '@/config/graphql-client';
import { useAuthStore } from '@/stores/useAuthStore';
import { useI18n } from '@/i18n/useI18n';
import { MainLayout } from '@/components/layout/MainLayout';
import { StandaloneLogin } from '@/components/auth/StandaloneLogin';
import { I18nProvider } from '@/i18n/I18nProvider';
import { resolveLocaleFromSearch } from '@/i18n/locale';
import { TooltipProvider } from '@/components/ui/tooltip';
import './globals.css';

const locale = resolveLocaleFromSearch(window.location.search);

function AppBootstrap() {
  const status = useAuthStore((state) => state.status);
  const error = useAuthStore((state) => state.error);
  const standaloneLoginDisabled = useAuthStore((state) => state.standaloneLoginDisabled);
  const { t } = useI18n();

  React.useEffect(() => {
    void (async () => {
      await useAuthStore.getState().initialize();
    })();
  }, []);

  if (status === 'loading') {
    return (
      <div
        className="flex h-screen items-center justify-center bg-background text-sm text-muted-foreground"
        data-testid="auth.bootstrap.loading"
        data-qa-module="auth"
        data-qa-object="session"
        data-qa-state="loading"
      >
        {t('common.auth.loading')}
      </div>
    );
  }

  if (status === 'error') {
    return (
      <div
        className="flex h-screen items-center justify-center bg-background p-6 text-center text-sm text-destructive"
        data-testid="auth.bootstrap.error"
        data-qa-module="auth"
        data-qa-object="session"
        data-qa-state="error"
        data-qa-error-code="bootstrap_failed"
      >
        {t('common.auth.bootstrapFailed', { message: error ?? t('common.unknownError') })}
      </div>
    );
  }

  if (status === 'unauthenticated') {
    if (standaloneLoginDisabled) {
      return (
        <div
          className="flex h-screen items-center justify-center bg-background p-6 text-center text-sm text-muted-foreground"
          data-testid="auth.standalone.disabled"
          data-qa-module="auth"
          data-qa-object="standalone-login"
          data-qa-state="disabled"
          data-qa-disabled-reason="standalone_login_disabled"
        >
          {t('standaloneLogin.disabled')}
        </div>
      );
    }
    return <StandaloneLogin />;
  }

  return <MainLayout />;
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <I18nProvider locale={locale}>
      <ApolloProvider client={graphqlClient}>
        <TooltipProvider>
          <AppBootstrap />
        </TooltipProvider>
      </ApolloProvider>
    </I18nProvider>
  </React.StrictMode>
);
