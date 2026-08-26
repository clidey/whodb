/**
 * ListCall is the thenable returned by paginated facade methods: `await`
 * yields the first page's rows; `.pages()` iterates all pages.
 */
export class ListCall {
    fetchPage;
    pageSize;
    constructor(fetchPage, pageSize) {
        this.fetchPage = fetchPage;
        this.pageSize = pageSize;
    }
    then(onfulfilled, onrejected) {
        return this.fetchPage(0).then(page => page.rows).then(onfulfilled, onrejected);
    }
    /** Iterates pages until a short page signals the end of the result set. */
    async *pages() {
        let offset = 0;
        for (;;) {
            const page = await this.fetchPage(offset);
            yield page;
            if (page.rows.length < this.pageSize)
                return;
            offset += this.pageSize;
        }
    }
    /** Iterates individual rows across all pages. */
    async *[Symbol.asyncIterator]() {
        for await (const page of this.pages()) {
            for (const row of page.rows)
                yield row;
        }
    }
}
