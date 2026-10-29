import React from "react";
import { AppProps } from "next/app";
import { MantineProvider } from "@mantine/core";
import { SocketProvider } from "../providers/WebSocket";
import { AppStoreProvider } from "../providers/AppStore";
import { resolver, theme } from "../lib/theme";

import "../styles/index.css";

function MyApp({ Component, pageProps, router }: AppProps) {
    return (
        <MantineProvider theme={theme} cssVariablesResolver={resolver} defaultColorScheme="auto">
            {router.pathname === "/admin" ? (
                // the admin page reads the admin API, not the game's websocket
                <Component {...pageProps} />
            ) : (
                <AppStoreProvider>
                    <SocketProvider>
                        <Component {...pageProps} />
                    </SocketProvider>
                </AppStoreProvider>
            )}
        </MantineProvider>
    );
}

export default MyApp;
