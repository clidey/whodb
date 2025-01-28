import MarkdownPreview from "@uiw/react-markdown-preview";
import React, { FC, useCallback, useEffect, useMemo, useRef, useState } from "react";
import ReactJson from "react-json-view";
import { useAppSelector } from "../store/hooks";
import { Icons } from "./icons";
import {basicSetup} from "codemirror";
import { json } from "@codemirror/lang-json";
import { markdown } from "@codemirror/lang-markdown";
import { sql } from "@codemirror/lang-sql";
import { EditorState } from "@codemirror/state";
import { EditorView, lineNumbers } from "@codemirror/view";
import { oneDark } from "@codemirror/theme-one-dark";
import classNames from "classnames";

type ICodeEditorProps = {
  value: string;
  setValue?: (value: string) => void;
  language?: "sql" | "markdown" | "json";
  onRun?: () => void;
  defaultShowPreview?: boolean;
  disabled?: boolean;
};

export const CodeEditor: FC<ICodeEditorProps> = ({
  value,
  setValue,
  language = "sql",
  onRun,
  defaultShowPreview = false,
  disabled,
}) => {
  const [showPreview, setShowPreview] = useState(defaultShowPreview);
  const editorRef = useRef<HTMLDivElement>(null);
  const darkModeEnabled = useAppSelector((state) => state.global.theme === "dark");
  const onRunReference = useRef<Function>();

  useEffect(() => {
    onRunReference.current = onRun;
  }, [onRun]);

  useEffect(() => {
    if (editorRef.current == null) {
        return;
    }

    const languageExtension = (() => {
      switch (language) {
        case "json":
          return json();
        case "markdown":
          return markdown();
        case "sql":
          return sql();
        default:
          return sql();
      }
    })();

    const state = EditorState.create({
        doc: value,
        extensions: [
          EditorView.domEventHandlers({
              keydown(event) {
                  if (event.metaKey && event.key === "Enter" && onRunReference.current != null) {
                      onRunReference.current();
                      event.preventDefault();
                      event.stopPropagation();
                  }
              },
          }),
            basicSetup,
            languageExtension,
            darkModeEnabled ? oneDark : [],
            EditorView.updateListener.of((update) => {
                if (update.changes && setValue != null) {
                    setValue(update.state.doc.toString());
                }
            }),
            lineNumbers(),
            EditorView.lineWrapping,
        ],
    });
 
    const view = new EditorView({
        state,
        parent: editorRef.current,
    });

    return () => {
      view.destroy();
    };
  }, [darkModeEnabled]);

  const handlePreviewToggle = useCallback(() => {
    setShowPreview((prev) => !prev);
  }, []);

  const hidePreview = useMemo(() => {
    return language !== "markdown" && language !== "json";
  }, [language]);

  const children = useMemo(() => {
    if (showPreview) {
      if (language === "markdown") {
        return (
          <div className="overflow-y-auto h-full bg-white p-4 pl-8 dark:bg-[#252526] dark:backdrop-blur-md">
            <MarkdownPreview
              className="pointer-events-none"
              source={value}
              wrapperElement={{
                "data-color-mode": darkModeEnabled ? "dark" : "light",
              }}
              style={{
                backgroundColor: "unset",
              }}
            />
          </div>
        );
      }
      if (language === "json") {
        return (
          <div className="overflow-y-auto h-full bg-white p-4 pl-8 dark:bg-[#252526] dark:backdrop-blur-md">
            <ReactJson
              src={JSON.parse(value)}
              theme={darkModeEnabled ? "bright" : undefined}
              style={{ height: "100%", backgroundColor: "unset" }}
            />
          </div>
        );
      }
    }
    return <div ref={editorRef} className="h-full w-full [&>.cm-editor]:h-full [&>.cm-editor]:p-2 dark:[&>.cm-editor]:bg-[#252526] dark:[&_.cm-gutter]:bg-[#252526]"></div>;
  }, [darkModeEnabled, showPreview, value, language]);

  const actionButtons = useMemo(() => {
    return (
      <button
        className="transition-all cursor-pointer hover:scale-110 hover:bg-gray-100/50 rounded-full p-1"
        onClick={handlePreviewToggle}
      >
        {React.cloneElement(showPreview ? Icons.Hide : Icons.Show, {
          className: "stroke-teal-500 w-8 h-8",
        })}
      </button>
    );
  }, [handlePreviewToggle, showPreview]);

  return (
    <div className={classNames("relative h-full w-full", {
        "opacity-50 pointer-events-none": disabled,
    })}>
      {children}
      <div
        className={classNames("absolute right-6 bottom-2 z-20", {
          hidden: hidePreview,
        })}
      >
        {actionButtons}
      </div>
    </div>
  );
};
