import type { Row } from './hydrate.js';

/** One page of hydrated rows. */
export interface Page {
  rows: Row[];
  totalCount: number | null;
  pageOffset: number;
}

/**
 * ListCall is the thenable returned by paginated facade methods: `await`
 * yields the first page's rows; `.pages()` iterates all pages.
 */
export class ListCall implements PromiseLike<Row[]> {
  private readonly fetchPage: (pageOffset: number) => Promise<Page>;
  private readonly pageSize: number;

  constructor(fetchPage: (pageOffset: number) => Promise<Page>, pageSize: number) {
    this.fetchPage = fetchPage;
    this.pageSize = pageSize;
  }

  then<TResult1 = Row[], TResult2 = never>(
    onfulfilled?: ((value: Row[]) => TResult1 | PromiseLike<TResult1>) | null,
    onrejected?: ((reason: unknown) => TResult2 | PromiseLike<TResult2>) | null,
  ): PromiseLike<TResult1 | TResult2> {
    return this.fetchPage(0).then(page => page.rows).then(onfulfilled, onrejected);
  }

  /** Iterates pages until a short page signals the end of the result set. */
  async *pages(): AsyncGenerator<Page> {
    let offset = 0;
    for (;;) {
      const page = await this.fetchPage(offset);
      yield page;
      if (page.rows.length < this.pageSize) return;
      offset += this.pageSize;
    }
  }

  /** Iterates individual rows across all pages. */
  async *[Symbol.asyncIterator](): AsyncGenerator<Row> {
    for await (const page of this.pages()) {
      for (const row of page.rows) yield row;
    }
  }
}
