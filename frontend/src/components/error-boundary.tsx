/*
 * Copyright 2026 Clidey, Inc.
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

import {Button} from "@clidey/ux";
import {Component, type ErrorInfo, type ReactNode} from "react";
import {captureException} from "../config/posthog";
import {openExternalLink} from "../utils/external-links";

interface ErrorBoundaryProps {
    children: ReactNode;
}

interface ErrorBoundaryState {
    hasError: boolean;
}

/**
 * Top-level error boundary that catches unhandled React component errors
 * and displays a fallback UI instead of a blank screen.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    state: ErrorBoundaryState = {hasError: false};

    static getDerivedStateFromError(): ErrorBoundaryState {
        return {hasError: true};
    }

    componentDidCatch(error: Error, info: ErrorInfo) {
        console.error("Uncaught error in component tree:", error, info.componentStack);
        captureException(error, {componentStack: info.componentStack ?? ""});
    }

    private handleGoHome = () => {
        window.location.href = "/";
    };

    render() {
        if (this.state.hasError) {
            return (
                <div className="flex flex-col items-center justify-center min-h-screen gap-6 p-8">
                    <h1 className="text-3xl font-bold text-foreground">
                        Whoops, something went wrong.
                    </h1>
                    <p className="text-muted-foreground">
                        Please either refresh the page or return home to try again.
                    </p>
                    <p className="text-muted-foreground">
                        If the issue continues, please{" "}
                        <a
                            href="https://github.com/clidey/whodb/issues"
                            onClick={(e) => openExternalLink("https://github.com/clidey/whodb/issues", e)}
                            className="text-primary underline underline-offset-2"
                        >
                            get in touch.
                        </a>
                    </p>
                    <Button onClick={this.handleGoHome}>Go home</Button>
                </div>
            );
        }
        return this.props.children;
    }
}
