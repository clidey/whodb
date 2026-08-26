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
export declare class ListCall implements PromiseLike<Row[]> {
    private readonly fetchPage;
    private readonly pageSize;
    constructor(fetchPage: (pageOffset: number) => Promise<Page>, pageSize: number);
    then<TResult1 = Row[], TResult2 = never>(onfulfilled?: ((value: Row[]) => TResult1 | PromiseLike<TResult1>) | null, onrejected?: ((reason: unknown) => TResult2 | PromiseLike<TResult2>) | null): PromiseLike<TResult1 | TResult2>;
    /** Iterates pages until a short page signals the end of the result set. */
    pages(): AsyncGenerator<Page>;
    /** Iterates individual rows across all pages. */
    [Symbol.asyncIterator](): AsyncGenerator<Row>;
}
