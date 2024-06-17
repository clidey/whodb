import classNames from "classnames";
import { clone } from "lodash";
import { ChangeEvent, FC, KeyboardEvent, useCallback, useMemo, useRef, useState } from "react";
import { twMerge } from "tailwind-merge";
import { toTitleCase } from "../utils/functions";
import { AnimatedButton } from "./button";
import { useExportToCSV } from "./hooks";
import { Icons } from "./icons";
import { SearchInput } from "./search";
 
type IPaginationProps = {
    pageCount: number;
    currentPage: number;
    onPageChange?: (page: number) => void;
}

const Pagination: FC<IPaginationProps> = ({ pageCount, currentPage, onPageChange }) => {
    const renderPageNumbers = () => {
        const pageNumbers = [];
        const maxVisiblePages = 5;

        if (pageCount <= maxVisiblePages) {
            for (let i = 1; i <= pageCount; i++) {
                pageNumbers.push(
                    <div
                        key={i}
                        className={`cursor-pointer p-2 text-sm hover:scale-110 hover:bg-gray-200 rounded-md text-gray-600 ${currentPage === i ? 'bg-gray-300' : ''}`}
                        onClick={() => onPageChange?.(i)}
                    >
                        {i}
                    </div>
                );
            }
        } else {
            // Show first, last, current, and adjacent pages with ellipses
            const createPageItem = (i: number) => (
                <div
                    key={i}
                    className={`cursor-pointer p-2 text-sm hover:scale-110 hover:bg-gray-200 rounded-md text-gray-600 ${currentPage === i ? 'bg-gray-300' : ''}`}
                    onClick={() => onPageChange?.(i)}
                >
                    {i}
                </div>
            );

            pageNumbers.push(createPageItem(1));

            if (currentPage > 3) {
                pageNumbers.push(
                    <div key="start-ellipsis" className="cursor-default p-2 text-sm text-gray-600">...</div>
                );
            }

            const startPage = Math.max(2, currentPage - 1);
            const endPage = Math.min(pageCount - 1, currentPage + 1);

            for (let i = startPage; i <= endPage; i++) {
                pageNumbers.push(createPageItem(i));
            }

            if (currentPage < pageCount - 2) {
                pageNumbers.push(
                    <div key="end-ellipsis" className="cursor-default p-2 text-sm text-gray-600">...</div>
                );
            }

            pageNumbers.push(createPageItem(pageCount));
        }

        return pageNumbers;
    };

    return (
        <div className="flex space-x-2">
            {renderPageNumbers()}
        </div>
    );
};

type ITDataProps = {
    row: string[];
    data: string;
}

const TData: FC<ITDataProps> = ({ data, row }) => {
    const [editedData, setEditedData] = useState(data);
    const ref = useRef<HTMLTableCellElement>(null);
    const [editable, setEditable] = useState(false);

    const handleChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
        setEditedData(e.target.value);
    }, []);
    
    const handleCancel = useCallback(() => {
        setEditedData(data);
        setEditable(false);
    }, [data]);

    const handleEdit = useCallback(() => {
        setEditable(true);
    }, []);

    const handleUpdate = useCallback(() => {
        console.log("Update", row, data, ref.current?.innerText);
    }, [data, row]);

    return <td className="focus:outline-none group/data cursor-pointer transition-all text-xs table-cell  border-t border-l last:border-r group-last/row:border-b group-last/row:first:rounded-bl-lg group-last/row:last:rounded-br-lg border-gray-200 relative p-0 overflow-hidden">
        <span className="hidden">{editedData}</span>
        <input className={classNames("w-full h-full p-2 leading-tight focus:outline-none focus:shadow-outline appearance-none transition-all duration-300", {
            "group-even/row:bg-gray-200 hover:bg-gray-300 group-even/row:hover:bg-gray-300": !editable,
            "bg-transparent": editable,
        })} disabled={!editable} value={editedData} onChange={handleChange} />
        {
            editable &&
            <div className="transition-all hidden group-hover/data:flex absolute right-8 top-1/2 -translate-y-1/2 hover:scale-125" onClick={handleCancel}>
                {Icons.Cancel}
            </div>
        }
        <div className="transition-all hidden group-hover/data:flex absolute right-2 top-1/2 -translate-y-1/2 hover:scale-125" onClick={editable ? handleUpdate : handleEdit}>
            {editable ? Icons.CheckCircle : Icons.Edit}
        </div>
    </td>
}

const TableRow: FC<{ rowIndex: number, row: string[] }> = ({ rowIndex, row }) => {
    return (
        <tr className="text-xs group/row">
            {
                row.map((datum, columnIndex) => (
                    <TData key={`data-${rowIndex}-${columnIndex}`} data={datum} row={row} />
                ))
            }
        </tr>
    )
}

type ITableProps = {
    className?: string;
    columns: string[];
    columnTags?: string[];
    rows: string[][];
    totalPages: number;
    currentPage: number;
    onPageChange?: (page: number) => void;
}

export const Table: FC<ITableProps> = ({ className, columns, rows, columnTags, totalPages, currentPage, onPageChange }) => {
    const tableRef = useRef<HTMLTableElement>(null);
    const [direction, setDirection] = useState<"asc" | "dsc">();
    const [sortedColumn, setSortedColumn] = useState<string>();
    const [search, setSearch] = useState("");
    const [searchIndex, setSearchIndex] = useState(0);

    const rowCount = useMemo(() => {
        return rows.length ?? 0;
    }, [rows]);

    const handleKeyUp = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
        if (tableRef.current == null) {
            return;
        }
        let interval: NodeJS.Timeout;
        if (e.key === "Enter") {
            let newSearchIndex = (searchIndex+1) % rowCount;
            setSearchIndex(newSearchIndex);
            const searchText = search.toLowerCase();
            let index = 0;
            const tbody = tableRef.current.querySelector("tbody");
            if (tbody == null) {
                return;
            }
            for (const childNode of tbody.childNodes) {
                if (childNode instanceof HTMLTableRowElement) {
                    const text = childNode.textContent?.toLowerCase();
                    if (text != null && searchText != null && text.includes(searchText)) {
                        if (index === newSearchIndex) {
                            childNode.scrollIntoView({
                                behavior: "smooth",
                                block: "center",
                                inline: "center",
                            });
                            for (const cell of childNode.querySelectorAll("input")) {
                                if (cell instanceof HTMLInputElement) {
                                    cell.classList.add("!bg-yellow-100");
                                    interval = setTimeout(() => {
                                        cell.classList.remove("!bg-yellow-100");
                                    }, 3000);
                                }
                            }
                            return;
                        }
                        index++;
                    }
                }
            };
        }
        
        return () => {
            if (interval != null) {
                clearInterval(interval);
            }
        }
    }, [search, rowCount, searchIndex]);

    const handleSearchChange = useCallback((newValue: string) => {
        setSearchIndex(-1);
        setSearch(newValue);
    }, []);

    const handleSort = useCallback((columnToSort: string) => {
        const columnSelectedIsDifferent = columnToSort !== sortedColumn;
        if (!columnSelectedIsDifferent && direction === "dsc") {
            setDirection(undefined);
            return setSortedColumn(undefined);
        }
        setSortedColumn(columnToSort);
        if (direction == null || columnSelectedIsDifferent) {
            return setDirection("asc");
        }
        setDirection("dsc");
    }, [sortedColumn, direction]);
    
    const sortedRows = useMemo(() => {
        if (sortedColumn == null) {
            return rows;
        }
        const columnIndex = columns.indexOf(sortedColumn);
        const newRows = clone(rows);
        newRows.sort((a, b) => {
            if (a[columnIndex] < b[columnIndex]) {
                return direction === 'asc' ? -1 : 1;
            }
            if (a[columnIndex] > b[columnIndex]) {
                return direction === 'asc' ? 1 : -1;
            }
            return 0;
        });
        return newRows;
    }, [sortedColumn, columns, direction, rows]);

    const exportToCSV = useExportToCSV(columns, rows);

    return (
        <div className="flex flex-col grow gap-4 items-center">
            <div className="flex justify-between items-center w-full">
                <div>
                    <SearchInput search={search} setSearch={handleSearchChange} placeholder="Search through rows     [Press Enter]" inputProps={{
                        className: "w-[300px]",
                        onKeyUp: handleKeyUp,
                    }} />
                </div>
                <div className="flex gap-4 items-center">
                    <div className="text-sm text-gray-600"><span className="font-semibold">Count:</span> {rowCount}</div>
                    <AnimatedButton icon={Icons.Download} label="Export" type="lg" onClick={exportToCSV} />
                </div>
            </div>
            <div className={twMerge(classNames("flex h-[60vh] grow flex-col gap-4 overflow-auto w-[80vw]", className))}>
                <table className="table-auto border-separate border-spacing-0 mt-4 h-fit w-full" ref={tableRef}>
                    <thead>
                        <tr>
                            {
                                columns.map((column, i) => (                        
                                    <th key={`column-name-${column}`} className="min-w-[150px] text-xs border-t border-l last:border-r border-gray-200 p-2 text-left bg-gray-500 text-white first:rounded-tl-lg last:rounded-tr-lg relative group/header cursor-pointer"
                                        onClick={() => handleSort(column)}>
                                        {toTitleCase(column)} [<span className="text-[11px]">{columnTags?.[i]}]</span>
                                        <div className={twMerge(classNames("transition-all absolute top-2 right-2 opacity-0", {
                                            "opacity-100": sortedColumn === column,
                                            "rotate-180": direction === "dsc",
                                        }))}>
                                            {Icons.ArrowUp}
                                        </div>
                                    </th>
                                ))
                            }
                        </tr>
                    </thead>
                    <tbody>
                        {
                            sortedRows.map((row, index) => (
                                <TableRow key={`row-${row[0]}`} row={row} rowIndex={index} />
                            ))
                        }
                    </tbody>
                </table>
            </div>
            <div className="flex justify-center items-center">
                <Pagination pageCount={totalPages} currentPage={currentPage} onPageChange={onPageChange} />
            </div>
        </div>
    )
}