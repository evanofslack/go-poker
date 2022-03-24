import Layout from "../components/Layout";
import Chat from "../components/Chat";
import { useEffect } from "react";

export default function IndexPage() {
    useEffect(() => {
    }, []);

    return (
        <Layout title="Poker">
            <Chat/>
        </Layout>
    );
}
