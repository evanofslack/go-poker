import CommunityCards from "./CommunityCards";
import Pot from "./Pot";

export default function Felt() {
    return (
        <div className="flex h-[120%] min-w-[120%] flex-col items-center justify-center rounded-full border border-8 border-teal-900 bg-emerald-700">
            <Pot />
            <div className="mt-4 mb-12 flex w-full items-center justify-center">
                <CommunityCards />
            </div>
        </div>
    );
}
