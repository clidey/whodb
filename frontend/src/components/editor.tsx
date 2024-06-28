import { FC, cloneElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import MonacoEditor, { EditorDidMount, monaco } from 'react-monaco-editor';
import MarkdownPreview from "@uiw/react-markdown-preview";
import classNames from "classnames";
import { Icons } from "./icons";

type ICodeEditorProps = {
    value: string;
    setValue: (value: string) => void;
    language?: "sql" | "markdown" | "json";
    options?: monaco.editor.IStandaloneEditorConstructionOptions;
    onRun?: () => void;
    defaultShowPreview?: boolean;
}

export const CodeEditor: FC<ICodeEditorProps> = ({ value, setValue, language, options = {}, onRun, defaultShowPreview }) => {
    const [previousValue, setPreviousValue] = useState<string>();
    const [showPreview, setShowPreview] = useState(defaultShowPreview);
    const editorRef = useRef<monaco.editor.IStandaloneCodeEditor>();

    const handleEditorDidMount: EditorDidMount = useCallback(editor => {
        editorRef.current = editor;
    }, []);

    const handlePreviewToggle = useCallback(() => {
        const shouldShowPreview = !showPreview;
        setShowPreview(shouldShowPreview);
        if (language === "json") {
            if (shouldShowPreview) {
                setPreviousValue(value);
                return editorRef.current?.getAction('editor.action.formatDocument')?.run();
            }
            if (previousValue != null) {
                editorRef.current?.getModel()?.setValue(previousValue);
                return setPreviousValue(undefined);
            }
        }
    }, [language, previousValue, showPreview, value]);

    useEffect(() => {
        if (editorRef.current == null) {
            return;
        }
        const disposable = editorRef.current.onKeyDown(e => {
            if (e.metaKey && e.keyCode === monaco.KeyCode.Enter) {
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

    const children = useMemo(() => {
        if (showPreview) {
            if (language === "markdown") {
                return <div className="overflow-y-auto h-full bg-white p-4 pl-8">
                    <MarkdownPreview className="pointer-events-none" source={value} wrapperElement={{
                        "data-color-mode": "light",
                    }} />
                </div>
            }
        }
        return <MonacoEditor
            className={classNames({
                "pointer-events-none": showPreview,
                "pointer-events-auto": !showPreview,
            })}
            height="100%"
            width="100%"
            language={language}
            value={value}
            onChange={setValue}
            options={{
                fontSize: 12,
                glyphMargin: false,
                automaticLayout: true,
                selectOnLineNumbers: true,
                ...options,
            }}
            editorDidMount={handleEditorDidMount}
        />;
    }, [handleEditorDidMount, language, options, setValue, showPreview, value]);

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