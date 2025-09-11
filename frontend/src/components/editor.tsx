/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {useTheme} from "@clidey/ux";
import {json} from "@codemirror/lang-json";
import {markdown} from "@codemirror/lang-markdown";
import {sql} from "@codemirror/lang-sql";
import {EditorState, RangeSet} from "@codemirror/state";
import {oneDark} from "@codemirror/theme-one-dark";
import {EditorView, gutter, GutterMarker, lineNumbers} from "@codemirror/view";
import {EyeIcon, EyeSlashIcon} from "@heroicons/react/24/outline";
import classNames from "classnames";
import {basicSetup} from "codemirror";
import React, {FC, useCallback, useEffect, useMemo, useRef, useState} from "react";
import ReactJson from "react-json-view";
import MarkdownPreview from 'react-markdown';
import remarkGfm from "remark-gfm";
import {useApolloClient} from "@apollo/client";
import {createSQLAutocomplete} from "./editor-autocomplete";

// SQL validation function
const isValidSQLQuery = (text: string): boolean => {
  const trimmed = text.trim();
  if (!trimmed) return false;
  
  // Basic SQL validation - check for common SQL keywords at the start
  const sqlKeywords = [
    'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER', 
    'WITH', 'EXPLAIN', 'DESCRIBE', 'SHOW', 'USE', 'SET'
  ];
  
  const upperText = trimmed.toUpperCase();
  return sqlKeywords.some(keyword => upperText.startsWith(keyword));
};

// Find all valid SQL queries and their starting line numbers
const findValidQueriesWithPositions = (doc: any): Array<{query: string, startLine: number}> => {
  const fullText = doc.toString();
  const lines = fullText.split('\n');
  const results: Array<{query: string, startLine: number}> = [];
  
  let currentQuery = '';
  let queryStartLine = 1;
  let inQuery = false;
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmedLine = line.trim();
    
    // Skip empty lines unless we're already in a query
    if (!trimmedLine && !inQuery) {
      continue;
    }
    
    // If this line starts a new query (contains SQL keywords)
    if (!inQuery && isValidSQLQuery(trimmedLine)) {
      currentQuery = trimmedLine;
      queryStartLine = i + 1; // Convert to 1-based line number
      inQuery = true;
    }
    // If we're in a query, append to current query
    else if (inQuery) {
      currentQuery += '\n' + line;
    }
    
    // Check if this line ends the current query (contains semicolon)
    if (inQuery && line.includes(';')) {
      // Remove the semicolon and trim for validation
      const queryWithoutSemicolon = currentQuery.replace(/;$/, '').trim();
      
      if (isValidSQLQuery(queryWithoutSemicolon)) {
        results.push({
          query: queryWithoutSemicolon,
          startLine: queryStartLine
        });
      }
      
      // Reset for next query
      currentQuery = '';
      inQuery = false;
    }
  }
  
  // Handle case where there's no semicolon at the end
  if (inQuery && currentQuery.trim()) {
    const queryWithoutSemicolon = currentQuery.trim();
    if (isValidSQLQuery(queryWithoutSemicolon)) {
      results.push({
        query: queryWithoutSemicolon,
        startLine: queryStartLine
      });
    }
  }
  
  return results;
};

type ICodeEditorProps = {
  value: string;
  setValue?: (value: string) => void;
  language?: "sql" | "markdown" | "json";
  onRun?: (lineText?: string) => void;
  defaultShowPreview?: boolean;
  disabled?: boolean;
};

// Custom gutter marker for play button
class PlayButtonMarker extends GutterMarker {
  constructor(private onRun: (lineText?: string) => void, private queryText: string) {
    super();
  }

  toDOM() {
    const button = document.createElement("div");
    button.className = "cm-play-button";

    const svgNS = "http://www.w3.org/2000/svg";
    const svg = document.createElementNS(svgNS, "svg");
    svg.setAttribute("xmlns", svgNS);
    svg.setAttribute("fill", "none");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("stroke-width", "1.5");
    svg.setAttribute("stroke", "currentColor");
    svg.setAttribute("class", "w-4 h-4");

    const path = document.createElementNS(svgNS, "path");
    path.setAttribute("stroke-linecap", "round");
    path.setAttribute("stroke-linejoin", "round");
    path.setAttribute(
        "d",
        "M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 0 1 0 1.972l-11.54 6.347c-.75.412-1.667-.13-1.667-.986V5.653Z"
    );

    svg.appendChild(path);
    button.appendChild(svg);
    
    button.addEventListener("mouseenter", () => {
      button.style.opacity = "1";
    });
    
    button.addEventListener("mouseleave", () => {
      button.style.opacity = "0.6";
    });
    
    button.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      this.onRun(this.queryText);
    });
    
    return button;
  }
}

export const CodeEditor: FC<ICodeEditorProps> = ({
  value,
  setValue,
  language,
  onRun,
  defaultShowPreview = false,
  disabled,
}) => {
  const [showPreview, setShowPreview] = useState(defaultShowPreview);
  const editorRef = useRef<HTMLDivElement>(null);
  const onRunReference = useRef<Function>();
  const darkModeEnabled = useTheme().theme === "dark";
  const apolloClient = useApolloClient();

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
      }
    })();

    // Create custom gutter for SQL queries
    const createPlayButtonGutter = () => {
      if (language !== "sql" || !onRun) {
        return [];
      }

      return [
        gutter({
          class: "cm-play-gutter",
          markers: (view) => {
            const doc = view.state.doc;
            const ranges = [];
            
            // Find all valid queries with their starting positions
            const validQueries = findValidQueriesWithPositions(doc);
            
            // Add play buttons for each valid query
            for (const { query, startLine } of validQueries) {
              const startLineObj = doc.line(startLine);
              if (startLineObj && startLineObj.text.trim().length > 0) {
                // Create a unique marker for each query with the specific query text
                const playMarker = new PlayButtonMarker((lineText) => {
                  if (onRunReference.current) {
                    onRunReference.current(lineText);
                  }
                }, query);
                
                ranges.push({ from: startLineObj.from, to: startLineObj.from, value: playMarker });
              }
            }
            
            return RangeSet.of(ranges);
          },
        }),
      ];
    };

    const state = EditorState.create({
        doc: value,
        extensions: [
          EditorView.domEventHandlers({
              keydown(event) {
                if ((event.metaKey || event.ctrlKey) && event.key === "Enter" && onRunReference.current != null) {
                      // Get the selected text if any, otherwise use the entire content
                      const selection = view.state.selection;
                      let textToExecute = '';
                      
                      if (selection.main.empty) {
                        // No selection, execute entire content
                        textToExecute = view.state.doc.toString();
                      } else {
                        // Has selection, execute only the selected text
                        textToExecute = view.state.sliceDoc(selection.main.from, selection.main.to);
                      }
                      
                      onRunReference.current(textToExecute);
                      event.preventDefault();
                      event.stopPropagation();
                  }
              },
          }),
            basicSetup,
            languageExtension != null ? languageExtension : [],
          // Add autocomplete for SQL
          language === "sql" ? createSQLAutocomplete({apolloClient}) : [],
            darkModeEnabled ? [oneDark, EditorView.theme({
              ".cm-activeLine": { backgroundColor: "rgba(0,0,0,0.05) !important" },
              ".cm-activeLineGutter": { backgroundColor: "rgba(0,0,0,0.05) !important" },
              ".cm-play-gutter": { 
                width: "24px",
                backgroundColor: "transparent",
                borderRight: "1px solid rgba(0,0,0,0.1)",
              },
              ".cm-play-button": {
                color: "#10b981", // teal-500
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                height: "100%",
              },
              ".dark .cm-play-button": {
                color: "#14b8a6", // teal-400 for dark mode
              },
              ".dark .cm-play-gutter": {
                borderRight: "1px solid rgba(255,255,255,0.1)",
              },
            })] : [EditorView.theme({
              ".cm-activeLine": { backgroundColor: "rgba(0,0,0,0.05) !important" },
              ".cm-activeLineGutter": { backgroundColor: "rgba(0,0,0,0.05) !important" },
              ".cm-play-gutter": { 
                width: "24px",
                backgroundColor: "transparent",
                borderRight: "1px solid rgba(0,0,0,0.1)",
              },
              ".cm-play-button": {
                color: "#10b981", // teal-500
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                height: "100%",
              },
            })],
            EditorView.updateListener.of((update) => {
              if (update.docChanged && update.changes && setValue != null) {
                  setValue(update.state.doc.toString());
              }
            }),
            lineNumbers(),
            EditorView.lineWrapping,
            createPlayButtonGutter(),
        ],
    });
 
    const view = new EditorView({
        state,
        parent: editorRef.current,
    });

    return () => {
      view.destroy();
    };
  }, [language, apolloClient, darkModeEnabled]);

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
          <div className="h-full bg-white p-4 pl-8 dark:bg-[#252526] dark:backdrop-blur-md markdown-preview dark:*:text-neutral-300 overflow-y-auto">
            {/* todo: there seems to be an issue with links in markdown with the library */}
            <MarkdownPreview remarkPlugins={[remarkGfm]}>{value}</MarkdownPreview>
          </div>
        );
      }
      if (language === "json") {
        return (
          <div className="h-full bg-white p-4 pl-8 dark:bg-[#252526] dark:backdrop-blur-md overflow-y-auto">
            <ReactJson
              src={JSON.parse(value)}
              theme={darkModeEnabled ? "bright" : undefined}
              style={{ height: "100%", backgroundColor: "unset" }}
            />
          </div>
        );
      }
    }
    return null;
  }, [darkModeEnabled, showPreview, value, language]);

  const actionButtons = useMemo(() => {
    return (
      <button
        className="transition-all cursor-pointer hover:scale-110 hover:bg-gray-100/50 rounded-full p-1"
        onClick={handlePreviewToggle}
      >
        {React.cloneElement(showPreview ? <EyeSlashIcon className="w-4 h-4" /> : <EyeIcon className="w-4 h-4" />, {
          className: "stroke-teal-500 w-8 h-8",
        })}
      </button>
    );
  }, [handlePreviewToggle, showPreview]);

  return (
    <div className={classNames("relative h-full w-full", {
      "pointer-events-none": disabled,
    })}>
      {children}
      <div ref={editorRef} className={classNames("h-full w-full [&>.cm-editor]:h-full [&>.cm-editor]:p-2 [&>.cm-editor]:!bg-neutral-100 [&_.cm-gutters]:!bg-neutral-100 dark:[&>.cm-editor]:!bg-[#252526] dark:[&_.cm-gutters]:!bg-[#252526] transition-all opacity-100", {
        "opacity-0 pointer-events-none": hidePreview && disabled,
        }
      )} data-testid="code-editor"></div>
      <div
        className={classNames("absolute right-6 bottom-2 z-20", {
          hidden: hidePreview,
        })}>
        {actionButtons}
      </div>
    </div>
  );
};

