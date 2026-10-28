import React from "react";
import { AppProps } from "next/app";
import { MantineProvider } from "@mantine/core";
import { SocketProvider } from "../providers/WebSocket";
import { AppStoreProvider } from "../providers/AppStore";
import { resolver, theme } from "../lib/theme";

import "../styles/index.css";

function MyApp({ Component, pageProps }: AppProps) {
    return (
        <MantineProvider theme={theme} cssVariablesResolver={resolver} defaultColorScheme="dark">
            <AppStoreProvider>
                <SocketProvider>
                    <Component {...pageProps} />
                </SocketProvider>
            </AppStoreProvider>
        </MantineProvider>
    );
}

export default MyApp;
