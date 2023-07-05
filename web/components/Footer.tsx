import { FiGithub } from "react-icons/fi";

export default function Footer() {
    return (
        <footer className=" mt-10 flex w-screen flex-row items-center justify-between py-3 px-8 pb-4 text-sm font-light text-neutral-500 md:mt-20">
            <p> &copy; 2022 go-poker </p>
            <a href="https://github.com/evanofslack/go-poker">
                <FiGithub size="1.2rem" />
            </a>
        </footer>
    );
}
