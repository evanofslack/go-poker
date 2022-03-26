import React, { ReactNode, Fragment } from "react";
import MobileNav from "./MobileNav";
import Head from "next/head";
import Footer from "./Footer";

type Props = {
    children?: ReactNode;
    title?: string;
};

const Layout = ({ children, title = "page" }: Props) => (
    <Fragment>
        <Head>
            <title>{title}</title>
            <meta charSet="utf-8" />
            <meta name="viewport" content="initial-scale=1.0, width=device-width" />
        </Head>
        <main>{children}</main>
    </Fragment>
);

export default Layout;
