import { describe, expect, it } from 'vitest'

import { formatGraphQLErrorMessage } from '@/utils/graphql-error'

describe('formatGraphQLErrorMessage', () => {
  it('includes gqlgen error messages from Apollo ServerError results', () => {
    const error = Object.assign(
      new Error('Response not successful: Received status code 422'),
      {
        name: 'ServerError',
        statusCode: 422,
        result: {
          errors: [
            { message: 'Cannot query field "ReplaceRow" on type "Mutation".' },
          ],
          data: null,
        },
      },
    )

    expect(formatGraphQLErrorMessage(error)).toBe(
      'Response not successful: Received status code 422\nCannot query field "ReplaceRow" on type "Mutation".',
    )
  })

  it('keeps ordinary Error messages unchanged', () => {
    expect(formatGraphQLErrorMessage(new Error('update failed'))).toBe('update failed')
  })

  it('includes string server response bodies', () => {
    const error = Object.assign(new Error('Response not successful: Received status code 422'), {
      result: 'invalid request body',
    })

    expect(formatGraphQLErrorMessage(error)).toBe(
      'Response not successful: Received status code 422\ninvalid request body',
    )
  })
})
