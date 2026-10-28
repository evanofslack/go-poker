import { Html, Head, Main, NextScript } from "next/document";
import { ColorSchemeScript, mantineHtmlProps } from "@mantine/core";

export default function Document() {
    return (
        <Html {...mantineHtmlProps}>
            <Head>
                <ColorSchemeScript defaultColorScheme="dark" />
            </Head>
            <body className="m-0 bg-surface p-0">
                <Main />
                <NextScript />
            </body>
        </Html>
    );
}
