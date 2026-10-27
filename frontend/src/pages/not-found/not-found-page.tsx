import type {FC} from 'react';
import {Link} from 'react-router-dom';
import {Button} from '@clidey/ux';
import {useTranslation} from '@/hooks/use-translation';
import {getSurfaceFallbackPath} from '@/config/route-registry';
import {NotFoundMark} from '@/components/not-found/not-found-mark';

/**
 * Catch-all page for addresses that match no route. Rendered inside the
 * authenticated shell, both for unknown top-level paths and for unknown pages
 * under a valid workspace. The link goes to the edition's home, which EE
 * resolves to the workspace dashboard.
 */
export const NotFoundPage: FC = () => {
  const {t} = useTranslation('pages/not-found');
  return (
    <main className="whodb-404" data-testid="not-found-page">
      <NotFoundMark label={t('sceneLabel')} />
      <p className="whodb-404-eyebrow">{t('eyebrow')}</p>
      <h1 className="whodb-404-title">{t('title')}</h1>
      <p className="whodb-404-copy">{t('description')}</p>
      <Button asChild>
        <Link to={getSurfaceFallbackPath()}>{t('goHome')}</Link>
      </Button>
    </main>
  );
};

export default NotFoundPage;
