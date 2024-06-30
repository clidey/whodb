import MonacoEditor, { EditorProps, OnMount } from "@monaco-editor/react";
import MarkdownPreview from "@uiw/react-markdown-preview";
import classNames from "classnames";
import { KeyCode, editor, languages } from "monaco-editor";
import { FC, cloneElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Icons } from "./icons";
import { Loading } from "./loading";
import ReactJson from 'react-json-view';

languages.register({ id: 'markdown' });
languages.register({ id: 'json' });
languages.register({ id: 'sql' });

type ICodeEditorProps = {
    value: string;
    setValue: (value: string) => void;
    language?: "sql" | "markdown" | "json";
    options?: EditorProps["options"];
    onRun?: () => void;
    defaultShowPreview?: boolean;
    disabled?: boolean;
}

export const CodeEditor: FC<ICodeEditorProps> = ({ value, setValue, language, options = {}, onRun, defaultShowPreview, disabled }) => {
    const [showPreview, setShowPreview] = useState(defaultShowPreview);
    const editorRef = useRef<editor.IStandaloneCodeEditor>();

    const handleEditorDidMount: OnMount = useCallback(editor => {
        editorRef.current = editor;
    }, []);

    const handlePreviewToggle = useCallback(async () => {
        setShowPreview(p => !p);
    }, []);

    useEffect(() => {
        if (editorRef.current == null) {
            return;
        }
        const disposable = editorRef.current.onKeyDown(e => {
            if (e.metaKey && e.keyCode === KeyCode.Enter) {
                onRun?.();
            }
        });
        return () => {
            disposable.dispose();
        }
    }, [editorRef, onRun]);

    const hidePreview = useMemo(() => {
        return language !== "markdown" && language !== "json";
    }, [language]);

    const handleChange = useCallback((newValue: string | undefined) => {
        if (newValue != null) {
            setValue(newValue);
        }
    }, [setValue]);

    const children = useMemo(() => {
        if (showPreview) {
            if (language === "markdown") {
                return <div className="overflow-y-auto h-full bg-white p-4 pl-8">
                    <MarkdownPreview className="pointer-events-none" source={value} wrapperElement={{
                        "data-color-mode": "light",
                    }} />
                </div>
            }
            if (language === "json") {
                return <div className="overflow-y-auto h-full bg-white p-4 pl-8">
                    <ReactJson src={JSON.parse(value)}  />
                </div>
            }
        }
        return <MonacoEditor
            className={classNames({
                "pointer-events-none": showPreview || disabled,
                "pointer-events-auto": !showPreview && !disabled,
            })}
            height="100%"
            width="100%"
            language={language}
            value={value}
            onChange={handleChange}
            loading={<div className="flex justify-center items-center h-full w-full">
                <Loading textClassName="text-white" />
            </div>}
            options={{
                fontSize: 12,
                glyphMargin: false,
                automaticLayout: true,
                selectOnLineNumbers: true,
                ...options,
            }}
            onMount={handleEditorDidMount}
        />;
    }, [disabled, handleChange, handleEditorDidMount, language, options, showPreview, value]);

    const actionButtons = useMemo(() => {
        return <button className="transition-all cursor-pointer hover:scale-110 hover:bg-gray-100/50 rounded-full p-1" onClick={handlePreviewToggle}>
            {cloneElement(showPreview ? Icons.Hide : Icons.Show, {
                className: "stroke-teal-500 w-8 h-8",
            })}
        </button>
    }, [handlePreviewToggle, showPreview]);

    return (
        <div className="relative h-full w-full">
            {children}
            <div className={classNames("absolute right-6 bottom-2 z-20", {
                "hidden": hidePreview,
            })}>
                {actionButtons}
            </div>
        </div>
    );
}
