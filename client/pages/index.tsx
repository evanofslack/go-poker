import Layout from "../components/Layout";
import Chat from "../components/Chat";
import Join from "../components/Join";
import { useEffect, useContext } from "react";
import { AppContext } from "../providers/AppStore";

export default function IndexPage() {
    const AppStore = useContext(AppContext);

    return <Layout title="Poker">{AppStore.state.username != "" ? <Chat /> : <Join />}</Layout>;
}
