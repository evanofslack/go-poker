import Layout from "../components/Layout";
import Game from "../components/Game";
import Join from "../components/Join";
import { useEffect, useContext } from "react";
import { AppContext } from "../providers/AppStore";

export default function IndexPage() {
    const AppStore = useContext(AppContext);

    return <Layout title="Poker">{AppStore.state.username != "" ? <Game /> : <Join />}</Layout>;
}
