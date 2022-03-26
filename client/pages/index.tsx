import Layout from "../components/Layout";
import Game from "../components/Game";
import Join from "../components/Join";
import { useContext } from "react";
import { AppContext } from "../providers/AppStore";

export default function IndexPage() {
    const { appState, dispatch } = useContext(AppContext);

    return <Layout title="Poker">{appState.username != "" ? <Game /> : <Join />}</Layout>;
}
