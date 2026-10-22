import { pipe } from 'fp-ts/lib/function'
import { matchW } from 'fp-ts/lib/TaskEither'
import { useAtomValue } from 'jotai'
import { useCallback, useEffect, useState } from 'react'
import { serverURL } from '../atoms/settings'
import { ffetch } from '../lib/httpClient'
import { useToast } from './toast'

const useFetch = <R>(resource: string) => {
  const base = useAtomValue(serverURL)

  const { pushMessage } = useToast()

  const [data, setData] = useState<R>()
  const [error, setError] = useState<string>()

  const fetcher = useCallback(() => pipe(
    ffetch<R>(`${base}${resource}`),
    matchW(
      (l) => {
        setError(l)
        pushMessage(l, 'error')
      },
      (r) => setData(r)
    )
  )(), [base, resource, pushMessage])

  useEffect(() => {
    fetcher()
  }, [fetcher])

  return { data, error, fetcher }
}

export default useFetch