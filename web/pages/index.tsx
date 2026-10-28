import Layout from "../components/Layout";
import Game from "../components/Game";
import Join from "../components/Join";
import { Notifications } from "@mantine/notifications";
import { useContext } from "react";
import { AppContext } from "../providers/AppStore";

export default function IndexPage() {
    const { appState, dispatch } = useContext(AppContext);

    return (
        <Layout title="Poker">
            <Notifications position="top-center" limit={3} autoClose={4000} />
            {appState.username ? <Game /> : <Join />}
        </Layout>
    );
}
