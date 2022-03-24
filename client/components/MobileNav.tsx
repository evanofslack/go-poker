import { AiOutlineMenu } from "react-icons/ai";
import { useState } from "react";
import { GrClose } from "react-icons/gr";

export default function MobileNav() {
    const [isOpen, setIsOpen] = useState(false);
    const toggle = () => {
        setIsOpen((value) => !value);
    };

    return (
        <div>
            <nav className="flex flex-row items-center justify-between py-3 px-8 w-full bg-white drop-shadow">
                <h1 className="text-3xl font-semibold">PokerGo</h1>
                {/* {!isOpen && <AiOutlineMenu size="1.8rem" onClick={toggle} />}
                {isOpen && <GrClose size="1.5rem" onClick={toggle} />} */}
            </nav>
        </div>
    );
}
