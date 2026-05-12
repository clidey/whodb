import * as React from 'react'
import type { LoginCredentials } from '@graphql'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { useI18n } from '@/i18n/useI18n'
import { useAuthStore } from '@/stores/useAuthStore'

type StandaloneDatabaseType = 'Postgres' | 'MySQL' | 'MongoDB' | 'Redis' | 'ClickHouse'

interface DatabaseDefaults {
  port: string
  database: string
}

interface StandaloneLoginFormProps {
  onSubmit: (credentials: LoginCredentials) => Promise<unknown>
}

const databaseTypes: StandaloneDatabaseType[] = ['Postgres', 'MySQL', 'MongoDB', 'Redis', 'ClickHouse']

const databaseDefaults: Record<StandaloneDatabaseType, DatabaseDefaults> = {
  Postgres: { port: '5432', database: 'postgres' },
  MySQL: { port: '3306', database: '' },
  MongoDB: { port: '27017', database: 'admin' },
  Redis: { port: '6379', database: '' },
  ClickHouse: { port: '9000', database: 'default' },
}

/** Renders the standalone login entry backed by the auth store. */
export function StandaloneLogin() {
  const createStandaloneSession = useAuthStore((state) => state.createStandaloneSession)
  return <StandaloneLoginForm onSubmit={createStandaloneSession} />
}

/** Renders the manual database connection form for standalone login. */
export function StandaloneLoginForm({ onSubmit }: StandaloneLoginFormProps) {
  const { t } = useI18n()
  const [type, setType] = React.useState<StandaloneDatabaseType>('Postgres')
  const [host, setHost] = React.useState('localhost')
  const [port, setPort] = React.useState(databaseDefaults.Postgres.port)
  const [username, setUsername] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [database, setDatabase] = React.useState(databaseDefaults.Postgres.database)
  const [portEdited, setPortEdited] = React.useState(false)
  const [databaseEdited, setDatabaseEdited] = React.useState(false)
  const [submitting, setSubmitting] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)

  const handleTypeChange = (nextType: StandaloneDatabaseType) => {
    setType(nextType)
    const defaults = databaseDefaults[nextType]
    if (!portEdited) {
      setPort(defaults.port)
    }
    if (!databaseEdited) {
      setDatabase(defaults.database)
    }
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      await onSubmit({
        Type: type,
        Hostname: host,
        Username: username,
        Password: password,
        Database: database,
        Advanced: [{ Key: 'Port', Value: port }],
      })
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : String(submitError))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <form
        className="w-full max-w-sm rounded-lg border border-border bg-card p-6 shadow-sm"
        onSubmit={handleSubmit}
      >
        <div className="mb-6">
          <h1 className="text-lg font-semibold text-foreground">{t('standaloneLogin.title')}</h1>
        </div>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="standalone-db-type">{t('standaloneLogin.databaseType')}</Label>
            <select
              id="standalone-db-type"
              className={cn(
                'border-input flex h-9 w-full rounded-md border bg-transparent px-3 py-1 text-sm shadow-xs outline-none transition-colors',
                'focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]',
              )}
              value={type}
              onChange={(event) => handleTypeChange(event.target.value as StandaloneDatabaseType)}
              disabled={submitting}
            >
              {databaseTypes.map((databaseType) => (
                <option key={databaseType} value={databaseType}>
                  {databaseType}
                </option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-[1fr_104px] gap-3">
            <div className="space-y-2">
              <Label htmlFor="standalone-host">{t('standaloneLogin.host')}</Label>
              <Input
                id="standalone-host"
                value={host}
                onChange={(event) => setHost(event.target.value)}
                disabled={submitting}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="standalone-port">{t('standaloneLogin.port')}</Label>
              <Input
                id="standalone-port"
                inputMode="numeric"
                value={port}
                onChange={(event) => {
                  setPortEdited(true)
                  setPort(event.target.value)
                }}
                disabled={submitting}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="standalone-username">{t('standaloneLogin.username')}</Label>
            <Input
              id="standalone-username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              disabled={submitting}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="standalone-password">{t('standaloneLogin.password')}</Label>
            <Input
              id="standalone-password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              disabled={submitting}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="standalone-database">{t('standaloneLogin.database')}</Label>
            <Input
              id="standalone-database"
              value={database}
              onChange={(event) => {
                setDatabaseEdited(true)
                setDatabase(event.target.value)
              }}
              disabled={submitting}
            />
          </div>
        </div>

        {error ? (
          <div className="mt-4 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {t('standaloneLogin.error', { message: error })}
          </div>
        ) : null}

        <Button className="mt-6 w-full" type="submit" disabled={submitting}>
          {submitting ? t('standaloneLogin.connecting') : t('standaloneLogin.connect')}
        </Button>
      </form>
    </div>
  )
}
