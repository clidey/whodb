import classNames from "classnames";
import { cloneElement, FC, KeyboardEventHandler, useCallback, useMemo, useRef, useState } from "react";
import { createDropdownItem, Dropdown, IDropdownItem } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { Input } from "../../components/input";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, GetAiChatQuery, useGetAiChatLazyQuery, useGetAiModelsQuery } from "../../generated/graphql";
import { notify } from "../../store/function";
import { useAppSelector } from "../../store/hooks";
import { chooseRandomItems } from "../../utils/functions";
import { chatExamples } from "./examples";
import { CodeEditor } from "../../components/editor";
import { ActionButton } from "../../components/button";

type TableData = GetAiChatQuery["AIChat"][0]["Result"];

const TablePreview: FC<{ data: TableData, text: string }> = ({ data, text }) => {
    const [showSQL, setShowSQL] = useState(false);

    const handleCodeToggle = useCallback(() => {
        setShowSQL(status => !status);
    }, []);

    return <div className="flex flex-col w-full group/table-preview gap-2 relative">
        <div className="absolute -top-3 -left-3 opacity-0 group-hover/table-preview:opacity-100 transition-all z-[1]">
            <ActionButton containerClassName="w-8 h-8" className="w-5 h-5" icon={cloneElement(showSQL ? Icons.Tables : Icons.Code, {
                className: "w-6 h-6 stroke-white",
            })} onClick={handleCodeToggle} />
        </div>
        <div className="flex items-center gap-4 overflow-hidden break-all leading-6 shrink-0 h-full w-full">
            {
                showSQL
                ? <div className="h-[150px] w-full">
                    <CodeEditor value={text} disabled={true} />
                </div>
                :   text.trim().startsWith("SELECT")
                    ? <Table className="h-[150px]" columns={data?.Columns.map(c => c.Name) ?? []} columnTags={data?.Columns.map(c => c.Type)}
                    rows={data?.Rows ?? []} totalPages={1} currentPage={1} disableEdit={true} hideActions={true} />
                    : <div className="bg-white/10 text-neutral-800 dark:text-neutral-300 rounded-lg p-2 flex gap-2">
                        Action Executed
                        {Icons.CheckCircle}
                    </div>
            }
        </div>
    </div>
}

type IChatMessage = {
    type: "message";
    text: string;
    isUserInput?: boolean;
} | {
    type: "sql";
    data: TableData;
    text: string;
    isUserInput?: true;
}

export const ChatPage: FC = () => {
    const [chats, setChats] = useState<IChatMessage[]>([]);
    const [currentModel, setCurrentModel] = useState("");
    const [query, setQuery] = useState("");
    const { data } = useGetAiModelsQuery({
        onCompleted(data) {
            if (data.AIModel.length > 0) {
                setCurrentModel(data.AIModel[0]);
            }
        },
    });
    const [getAIChat, { loading }] = useGetAiChatLazyQuery();
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const current = useAppSelector(state => state.auth.current);
    const schema = useAppSelector(state => state.database.schema);
    const [currentSearchIndex, setCurrentSearchIndex] = useState<number>();

    const handleAIModelChange = useCallback((item: IDropdownItem) => {
        setCurrentModel(item.id);
    }, []);

    const models = useMemo(() => {
        return data?.AIModel.map(model => createDropdownItem(model)) ?? [];
    }, [data?.AIModel]);

    const examples = useMemo(() => {
        return chooseRandomItems(chatExamples);
    }, []);

    const handleSubmitQuery = useCallback(() => {
        setChats(chats => [...chats, { type: "message", text: query, isUserInput: true, }]);
        setTimeout(() => {
            if (scrollContainerRef.current != null) {
                scrollContainerRef.current.scroll({
                    top: scrollContainerRef.current.scrollHeight,
                    behavior: "smooth",
                });
            }
        }, 250);
        getAIChat({
            variables: {
                query,
                model: currentModel,
                previousConversation: chats.map(chat => `${chat.isUserInput ? "User" : "System"}: ${chat.text}`).join("\n"),
                schema,
                type: current?.Type as DatabaseType,
            },
            onCompleted(data) {
                const systemChats: IChatMessage[] = data.AIChat.map(chat => {
                    if (chat.Type === "sql") {
                        return {
                            type: "sql",
                            text: chat.Text,
                            data: chat.Result,
                        }
                    }
                    return {
                        type: "message",
                        text: chat.Text,
                    }
                });
                setChats(chats => [...chats, ...systemChats]);
                setTimeout(() => {
                    if (scrollContainerRef.current != null) {
                        scrollContainerRef.current.scroll({
                            top: scrollContainerRef.current.scrollHeight,
                            behavior: "smooth",
                        });
                    }
                }, 250);
            },
            onError(error) {
                notify("Unable to query. Try again. "+error.message, "error");
            },
        });
        setQuery("");
    }, [chats, current?.Type, currentModel, getAIChat, query, schema]);

    const handleKeyUp: KeyboardEventHandler<HTMLInputElement> = useCallback((e) => {
        if (e.key === "ArrowUp") {
          const foundSearchIndex = currentSearchIndex != null ? currentSearchIndex - 1 : chats.length - 1;
          let searchIndex = foundSearchIndex;
    
          while (searchIndex >= 0) {
            if (chats[searchIndex].isUserInput) {
              setCurrentSearchIndex(searchIndex);
              setQuery(chats[searchIndex].text);
              return;
            }
            searchIndex--;
          }
    
          if (currentSearchIndex !== chats.length - 1) {
            searchIndex = chats.length - 1;
            while (searchIndex > foundSearchIndex) {
              if (chats[searchIndex].isUserInput) {
                setCurrentSearchIndex(searchIndex);
                setQuery(chats[searchIndex].text);
                return;
              }
              searchIndex--;
            }
          }
        }
    }, [chats, currentSearchIndex]);

    const handleSelectExample = useCallback((example: string) => {
        setQuery(example);
    }, []);

    return (
        <InternalPage routes={[InternalRoutes.Chat]}>
            <div className="flex flex-col justify-center items-center w-full h-full gap-2">
                <div className="flex justify-end w-full">
                    <Dropdown className="w-[200px]" value={createDropdownItem(currentModel)}
                        items={models}
                        onChange={handleAIModelChange}/>
                </div>
                <div className="flex bg-white/5 grow w-full rounded-xl overflow-hidden">
                    {
                        chats.length === 0
                        ? <div className="flex flex-col justify-center items-center w-full gap-8">
                            <img src="/images/logo.png" alt="clidey logo" className="w-auto h-16" />
                            <div className="flex flex-wrap justify-center items-center gap-4">
                                {
                                    examples.map((example, i) => (
                                        <div key={`chat-${i}`} className="flex flex-col gap-2 w-[150px] h-[200px] border border-white/10 rounded-3xl p-4 text-sm text-neutral-800 dark:text-neutral-300 cursor-pointer hover:scale-105 transition-all"
                                            onClick={() => handleSelectExample(example.description)}>
                                            {example.icon}
                                            {example.description}
                                        </div>
                                    ))
                                }
                            </div>
                        </div>
                        : <div className="h-full w-full py-8 max-h-[calc(80vh-5px)] overflow-y-scroll" ref={scrollContainerRef}>
                            <div className="flex justify-center w-full">
                                <div className="flex w-[max(65%,450px)] flex-col gap-2">
                                    {
                                        chats.map((chat, i) => {
                                            if (chat.type === "message") {
                                                return <div key={`chat-${i}`} className={classNames("flex items-center gap-4 overflow-hidden break-words leading-6 shrink-0", {
                                                    "self-end": chat.isUserInput,
                                                    "self-start": !chat.isUserInput,
                                                })}>
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput && <img src="/images/logo.png" alt="clidey logo" className="w-auto h-6" />}
                                                    <div className={classNames("text-neutral-800 dark:text-neutral-300 px-4 py-2 rounded-lg", {
                                                        "bg-white/10": chat.isUserInput,
                                                    })}>
                                                        {chat.text}
                                                    </div>
                                                </div>
                                            }
                                            return <TablePreview text={chat.text} data={chat.data} />
                                        })
                                    }
                                    { loading &&  <div className="flex w-full justify-end mt-4">
                                        <Loading containerClassName="flex-row w-fit ml-8" loadingText="Waiting for response" textClassName="text-sm text-neutral-800 dark:text-neutral-300" className="w-4 h-4" />
                                    </div> }
                                </div>
                            </div>
                        </div>
                    }
                </div>
                <div className="relative w-full">
                    <div className={classNames("absolute right-2 top-1/2 -translate-y-1/2 z-10 backdrop-blur-lg rounded-full cursor-pointer hover:scale-110 transition-all", {
                        "opacity-50": loading,
                    })} onClick={loading ? undefined : handleSubmitQuery}>
                        {cloneElement(Icons.ArrowUp, {
                            className: "w-5 h-5 stroke-neutral-800 dark:stroke-neutral-300",
                        })}
                    </div>
                    <Input value={query} setValue={setQuery} placeholder="Talk to me..." onSubmit={handleSubmitQuery} inputProps={{
                        disabled: loading,
                        onKeyUp: handleKeyUp,
                    }} />
                </div>
            </div>
        </InternalPage>
    )
}   