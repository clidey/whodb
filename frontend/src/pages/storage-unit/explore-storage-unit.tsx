import { useQuery } from "@apollo/client";
import classNames from "classnames";
import { ChangeEvent, FC, useCallback, useMemo, useRef, useState } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { AnimatedButton } from "../../components/button";
import { Icons } from "../../components/icons";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, GetStorageUnitRowsDocument, GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables, StorageUnit } from "../../generated/graphql";
import { toTitleCase } from "../../utils/functions";
import { InputWithlabel } from "../../components/input";

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
        console.log(row, data, ref.current?.innerText);
    }, [data, row]);

    return <td className="focus:outline-none group/data cursor-pointer transition-all text-xs table-cell  border-t border-l last:border-r group-last/row:border-b group-last/row:first:rounded-bl-lg group-last/row:last:rounded-br-lg border-gray-200 relative p-0 overflow-hidden">
        <input className={classNames("w-full h-full p-2 leading-tight focus:outline-none focus:shadow-outline appearance-none", {
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

export const ExploreStorageUnit: FC = () => {
    const unit: StorageUnit = useLocation().state?.unit;

    const { data: rows, loading } = useQuery<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>(GetStorageUnitRowsDocument, {
        variables: {
            type: DatabaseType.Postgres,
            storageUnit: unit.Name,
        },
    });

    const totalCount = useMemo(() => {
        return unit?.Attributes.find(attribute => attribute.Key === "Row Count")?.Value ?? 0;
    }, [unit]);

    if (unit == null) {
        return <Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />
    }

    if (loading) {
        return <InternalPage routes={[InternalRoutes.Dashboard.StorageUnit, InternalRoutes.Dashboard.ExploreStorageUnit]}>
            <Loading />
        </InternalPage>
    }

    return <InternalPage routes={[InternalRoutes.Dashboard.StorageUnit, InternalRoutes.Dashboard.ExploreStorageUnit]}>
        <div className="flex flex-col grow gap-4">
            <div className="flex items-center justify-between">
                <div className="flex gap-2 items-center">
                    <div className="text-xl font-bold mr-4">{unit.Name}</div>
                    <AnimatedButton icon={Icons.Download} label="Export" />
                </div>
                <div className="text-sm mr-4"><span className="font-semibold">Count:</span> {totalCount}</div>
            </div>
            <div className="flex gap-2 items-center">
                <InputWithlabel label="Page Size" value="10" />
                <InputWithlabel label="Page Offset" value="0" />
                <InputWithlabel label="Where Condition" value="" />
            </div>
            {
                rows != null &&
                <table className="table-auto border-separate border-spacing-0 mt-4">
                    <thead>
                        <tr>
                            {
                                rows.Row.Columns.map(column => (                        
                                    <th key={`column-name-${column.Name}`} className="text-xs border-t border-l last:border-r border-gray-200 p-2 text-left bg-gray-500 text-white first:rounded-tl-lg last:rounded-tr-lg">{toTitleCase(column.Name)} [<span className="text-[11px]">{column.Type}]</span></th>
                                ))
                            }
                        </tr>
                    </thead>
                    <tbody>
                        {
                            rows.Row.Rows.map((row, rowIndex) => (
                                <tr key={`row-${rowIndex}`} className="text-xs group/row">
                                    {
                                        row.map((datum, columnIndex) => (
                                            <TData key={`data-${rowIndex}-${columnIndex}`} data={datum} row={row} />
                                        ))
                                    }
                                </tr>
                            ))
                        }
                    </tbody>
                </table>
            }
        </div>
    </InternalPage>
}