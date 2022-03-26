import React from "react";
import { AppProps } from "next/app";
import { SocketProvider } from "../providers/WebSocket";
import { AppStoreProvider } from "../providers/AppStore";

import "../styles/index.css";

function MyApp({ Component, pageProps }: AppProps) {
    return (
        <AppStoreProvider>
            <SocketProvider>
                <Component {...pageProps} />
            </SocketProvider>
        </AppStoreProvider>
    );
}

export default MyApp;
