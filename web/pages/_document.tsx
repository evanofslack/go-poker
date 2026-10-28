import { Html, Head, Main, NextScript } from "next/document";
import { ColorSchemeScript, mantineHtmlProps } from "@mantine/core";
import { looksScript } from "../lib/prefs";

export default function Document() {
    return (
        <Html {...mantineHtmlProps}>
            <Head>
                <ColorSchemeScript defaultColorScheme="auto" />
                <script dangerouslySetInnerHTML={{ __html: looksScript }} />
            </Head>
            <body className="m-0 bg-surface p-0">
                <Main />
                <NextScript />
            </body>
        </Html>
    );
}
