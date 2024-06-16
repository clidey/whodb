import { FC, useCallback, useEffect, useRef } from "react";
import MonacoEditor, { EditorDidMount, monaco } from 'react-monaco-editor';

type ICodeEditorProps = {
    value: string;
    setValue: (value: string) => void;
    language: "sql";
    options?: monaco.editor.IStandaloneEditorConstructionOptions;
    onRun?: () => void;
}

export const CodeEditor: FC<ICodeEditorProps> = ({ value, setValue, language, options = {}, onRun }) => {
    const editorRef = useRef<monaco.editor.IStandaloneCodeEditor>();

    const handleEditorDidMount: EditorDidMount = useCallback(editor => {
        editorRef.current = editor;
    }, []);

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

    return (
        <MonacoEditor
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
        />
    );
}